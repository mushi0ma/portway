import 'package:flutter/material.dart';

import '../core/theme.dart';
import 'codex.dart';
import 'common.dart';
import 'home_screen.dart';

enum StateFilter { all, active, completed, failed, stopped }

/// Pure filter used by the history tab (and unit tests): text search over the
/// prompt preview, model, provider and state label; state and provider facets.
List<CodexRun> filterRuns(
  List<CodexRun> runs, {
  String query = '',
  StateFilter state = StateFilter.all,
  String? provider,
  String Function(String state)? stateText,
}) {
  final q = query.trim().toLowerCase();
  return runs.where((r) {
    final okState = switch (state) {
      StateFilter.all => true,
      StateFilter.active => r.active,
      StateFilter.completed => r.state == 'completed',
      StateFilter.failed => r.state == 'failed',
      StateFilter.stopped => r.state == 'stopped',
    };
    if (!okState) return false;
    if (provider != null && (r.provider ?? 'codex') != provider) return false;
    if (q.isEmpty) return true;
    final haystack = [
      r.preview,
      r.model ?? '',
      providerTitle(r.provider ?? 'codex'),
      stateText?.call(r.state) ?? r.state,
    ].join(' ').toLowerCase();
    return haystack.contains(q);
  }).toList();
}

/// UI-10 history: searchable and filterable list of tasks.
class TasksTab extends StatefulWidget {
  const TasksTab({super.key, required this.bridge, required this.nav});
  final BridgeController bridge;
  final ShellNav nav;
  @override
  State<TasksTab> createState() => _TasksTabState();
}

class _TasksTabState extends State<TasksTab> {
  final _query = TextEditingController();
  StateFilter _state = StateFilter.all;
  String? _provider;

  @override
  void initState() {
    super.initState();
    _query.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  void _reset() {
    _query.clear();
    setState(() {
      _state = StateFilter.all;
      _provider = null;
    });
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l;
    return Scaffold(
      appBar: AppBar(title: Text(l.tasks)),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => widget.nav.compose(),
        icon: const Icon(Icons.add_rounded),
        label: Text(l.newTask),
      ),
      body: ListenableBuilder(
        listenable: widget.bridge,
        builder: (context, _) {
          final b = widget.bridge, now = b.now();
          final providers = <String>{
            for (final p in b.status?.providers ?? const <AiProvider>[]) p.id,
            for (final r in b.runs)
              if (r.provider != null) r.provider!,
          }.toList();
          final shown = filterRuns(
            b.runs,
            query: _query.text,
            state: _state,
            provider: _provider,
            stateText: (s) => stateLabel(context, s),
          );
          final filtered =
              _query.text.isNotEmpty ||
              _state != StateFilter.all ||
              _provider != null;
          final labels = {
            StateFilter.all: l.filterAll,
            StateFilter.active: l.filterActive,
            StateFilter.completed: l.filterCompleted,
            StateFilter.failed: l.filterFailed,
            StateFilter.stopped: l.filterStopped,
          };
          return PageBody(
            onRefresh: () => b.refresh(),
            padding: const EdgeInsets.fromLTRB(
              Space.lg,
              Space.sm,
              Space.lg,
              96,
            ),
            children: [
              TextField(
                controller: _query,
                textInputAction: TextInputAction.search,
                decoration: InputDecoration(
                  hintText: l.searchHint,
                  prefixIcon: const Icon(Icons.search_rounded),
                  suffixIcon: _query.text.isEmpty
                      ? null
                      : IconButton(
                          tooltip: l.clearSearch,
                          icon: const Icon(Icons.close_rounded),
                          onPressed: _query.clear,
                        ),
                ),
              ),
              const SizedBox(height: Space.md),
              SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: Row(
                  children: [
                    for (final f in StateFilter.values)
                      Padding(
                        padding: const EdgeInsets.only(right: Space.sm),
                        child: ChoiceChip(
                          label: Text(labels[f]!),
                          selected: _state == f,
                          onSelected: (_) => setState(() => _state = f),
                        ),
                      ),
                  ],
                ),
              ),
              if (providers.length > 1) ...[
                const SizedBox(height: Space.sm),
                SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: Row(
                    children: [
                      Padding(
                        padding: const EdgeInsets.only(right: Space.sm),
                        child: ChoiceChip(
                          label: Text(l.allProviders),
                          selected: _provider == null,
                          onSelected: (_) => setState(() => _provider = null),
                        ),
                      ),
                      for (final p in providers)
                        Padding(
                          padding: const EdgeInsets.only(right: Space.sm),
                          child: ChoiceChip(
                            label: Text(providerTitle(p)),
                            selected: _provider == p,
                            onSelected: (_) => setState(
                              () => _provider = _provider == p ? null : p,
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ],
              Padding(
                padding: const EdgeInsets.only(top: Space.lg, bottom: Space.sm),
                child: Row(
                  children: [
                    Expanded(
                      child: Semantics(
                        liveRegion: true,
                        child: Text(
                          filtered
                              ? l.foundCount(shown.length, b.runs.length)
                              : l.totalCount(b.runs.length),
                          style: context.text.bodySmall,
                        ),
                      ),
                    ),
                    if (filtered)
                      TextButton(
                        onPressed: _reset,
                        child: Text(l.resetFilters),
                      ),
                  ],
                ),
              ),
              if (b.failure case final f?) ...[
                ErrorNotice(f, retry: () => b.refresh()),
                const SizedBox(height: Space.md),
              ],
              if (b.loading && b.runs.isEmpty)
                LinearProgressIndicator(semanticsLabel: l.loading),
              if (!b.loading && b.runs.isEmpty && b.failure == null)
                EmptyState(
                  icon: Icons.forum_outlined,
                  title: l.noTasks,
                  message: l.noTasksHint,
                )
              else if (!b.loading && shown.isEmpty && b.runs.isNotEmpty)
                EmptyState(
                  icon: Icons.search_off_rounded,
                  title: l.nothingFound,
                  message: l.nothingFoundHint,
                  action: OutlinedButton(
                    onPressed: _reset,
                    child: Text(l.resetFilters),
                  ),
                ),
              for (final run in shown) ...[
                RunTile(
                  run: run,
                  now: now,
                  onTap: () => widget.nav.openRun(run.id),
                ),
                const SizedBox(height: Space.sm),
              ],
            ],
          );
        },
      ),
    );
  }
}
