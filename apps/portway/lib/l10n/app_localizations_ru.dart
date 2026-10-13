// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Russian (`ru`).
class AppLocalizationsRu extends AppLocalizations {
  AppLocalizationsRu([String locale = 'ru']) : super(locale);

  @override
  String get appTitle => 'Portway';

  @override
  String get connectTitle => 'Подключение к компьютеру';

  @override
  String get connectHint =>
      'Укажите HTTPS-адрес Portway, запущенного на вашем компьютере.';

  @override
  String get connectWhereTitle => 'Где взять адрес?';

  @override
  String get connectWhere =>
      'Его показывает Portway на компьютере после запуска (Tailscale Funnel). Адрес localhost с телефона не работает.';

  @override
  String get apiUrl => 'Адрес сервера';

  @override
  String get urlHint => 'https://portway.example.ts.net';

  @override
  String get urlHelper => 'Только HTTPS, без пути и параметров';

  @override
  String get urlRequired => 'Введите адрес сервера';

  @override
  String get connect => 'Подключить';

  @override
  String get loginTitle => 'Вход';

  @override
  String loginAt(String installation) {
    return 'Войдите в «$installation», чтобы управлять ИИ на компьютере.';
  }

  @override
  String get email => 'Email';

  @override
  String get password => 'Пароль';

  @override
  String get name => 'Имя';

  @override
  String get setupProof => 'Код настройки';

  @override
  String get setupProofHelper =>
      'Значение PORTWAY_SETUP_TOKEN из файла .env на компьютере';

  @override
  String get setupProofRequired => 'Введите код настройки';

  @override
  String get login => 'Войти';

  @override
  String get noAccount => 'Нет аккаунта?';

  @override
  String get haveAccount => 'Уже есть аккаунт?';

  @override
  String get createAccount => 'Зарегистрироваться';

  @override
  String get register => 'Зарегистрироваться';

  @override
  String get registrationTitle => 'Регистрация владельца';

  @override
  String registrationHint(String installation) {
    return 'Владелец установки «$installation» — её единственный пользователь. После регистрации подтвердите email.';
  }

  @override
  String get repeatPassword => 'Повторите пароль';

  @override
  String get repeatRequired => 'Введите пароль ещё раз';

  @override
  String get passwordMismatch => 'Пароли не совпадают';

  @override
  String get ruleLength => 'Не менее 8 символов';

  @override
  String get ruleMix => 'Есть буква и цифра';

  @override
  String get nameRequired => 'Введите имя';

  @override
  String get nameTooLong => 'Не более 100 символов';

  @override
  String get emailRequired => 'Введите email';

  @override
  String get invalidEmail => 'Некорректный email. Пример: name@example.com';

  @override
  String get passwordRequired => 'Введите пароль';

  @override
  String get shortPassword => 'Пароль должен содержать не менее 8 символов';

  @override
  String get weakPassword => 'Добавьте хотя бы одну букву и одну цифру';

  @override
  String get confirmationTitle => 'Подтвердите email';

  @override
  String confirmationHint(String email) {
    return 'Мы отправили письмо на $email. Перейдите по ссылке из письма, затем войдите с тем же паролем.';
  }

  @override
  String get confirmationStepsTitle => 'Ссылка откроет приложение';

  @override
  String get confirmationSteps =>
      'Если письмо открыто на этом телефоне, Portway сам вернётся на экран входа. Проверьте папку «Спам», если письма нет.';

  @override
  String get editRegistration => 'Изменить данные';

  @override
  String get returnLogin => 'Перейти ко входу';

  @override
  String get change => 'Сменить';

  @override
  String get changeInstallation => 'Сменить установку';

  @override
  String get changeInstallationTitle => 'Сменить установку?';

  @override
  String get changeConfirm =>
      'Сессия на этом устройстве будет завершена. Для другой установки потребуется войти заново.';

  @override
  String get changeConfirmSignedOut =>
      'Приложение забудет адрес этой установки. Затем укажите адрес другого компьютера.';

  @override
  String get cancel => 'Отмена';

  @override
  String get continueAction => 'Продолжить';

  @override
  String get showPassword => 'Показать пароль';

  @override
  String get hidePassword => 'Скрыть пароль';

  @override
  String get credentialsTitle => 'Не удалось войти';

  @override
  String get credentialsError =>
      'Неверный email или пароль. Проверьте данные и попробуйте снова.';

  @override
  String get sessionExpiredTitle => 'Сессия завершена';

  @override
  String get sessionExpired => 'Срок входа истёк. Войдите снова.';

  @override
  String get deviceRevokedTitle => 'Доступ устройства отозван';

  @override
  String get deviceRevoked =>
      'Это устройство отключено от установки. Войдите снова.';

  @override
  String get signedOut => 'Вы вышли из аккаунта';

  @override
  String get signedOutHint =>
      'Разделы приложения снова доступны только после входа.';

  @override
  String get logoutPending =>
      'Выход выполнен на телефоне; сервер подтвердит отзыв доступа при следующем подключении.';

  @override
  String get registrationUnknownTitle => 'Ответ на регистрацию не получен';

  @override
  String get registrationUnknown =>
      'Проверьте почту. Повтор отправит тот же запрос и не создаст второй аккаунт.';

  @override
  String get retrySame => 'Повторить тот же запрос';

  @override
  String get home => 'Главная';

  @override
  String get tasks => 'Задачи';

  @override
  String get aiAccounts => 'Аккаунты ИИ';

  @override
  String get profile => 'Профиль';

  @override
  String get loading => 'Загрузка';

  @override
  String get retry => 'Повторить';

  @override
  String get refresh => 'Обновить';

  @override
  String get copy => 'Копировать';

  @override
  String get copied => 'Скопировано';

  @override
  String get open => 'Открыть';

  @override
  String get done => 'Готово';

  @override
  String get remove => 'Удалить';

  @override
  String get manage => 'Управление';

  @override
  String get details => 'Подробности';

  @override
  String get code => 'код';

  @override
  String get copyCode => 'Скопировать код';

  @override
  String get copyAnswer => 'Скопировать ответ';

  @override
  String greeting(String dayPart, String name) {
    return '$dayPart, $name';
  }

  @override
  String get morning => 'Доброе утро';

  @override
  String get day => 'Добрый день';

  @override
  String get evening => 'Добрый вечер';

  @override
  String get night => 'Доброй ночи';

  @override
  String get ownerFallback => 'владелец';

  @override
  String get computer => 'Компьютер';

  @override
  String get online => 'На связи';

  @override
  String get offline => 'Не в сети';

  @override
  String get statusUnknown => 'Связь проверяется';

  @override
  String get statusChecking => 'Запрашиваем состояние компьютера…';

  @override
  String computerOnlineHint(String workspace) {
    return 'Проект «$workspace» · только чтение';
  }

  @override
  String computerLastSeen(String ago) {
    return 'Был на связи $ago';
  }

  @override
  String get computerNeverSeen =>
      'Мост Portway на компьютере ещё не подключался';

  @override
  String get offlineHint =>
      'Запустите мост Portway на компьютере — статус обновится автоматически.';

  @override
  String get newTask => 'Новая задача';

  @override
  String get pendingTitle => 'Проверить отправку';

  @override
  String get aiModels => 'Модели ИИ';

  @override
  String get usageLoading => 'Загружаем аккаунты и лимиты…';

  @override
  String get providersPending =>
      'Компьютер передаст данные об аккаунтах и лимитах после обновления сервера и моста.';

  @override
  String get providersOffline =>
      'Данные об аккаунтах ИИ появятся, когда компьютер выйдет на связь.';

  @override
  String get providersEmptyTitle => 'Аккаунтов пока нет';

  @override
  String get recentTasks => 'Последние задачи';

  @override
  String get allTasks => 'Все задачи';

  @override
  String get noTasks => 'Задач пока нет';

  @override
  String get noTasksHint =>
      'Начните с вопроса о проекте — например, попросите объяснить его структуру.';

  @override
  String get available => 'Доступен';

  @override
  String get notInstalled => 'Не установлен';

  @override
  String get noLogin => 'Нет входа';

  @override
  String get limitExhausted => 'Лимит исчерпан';

  @override
  String get limitNear => 'Лимит почти исчерпан';

  @override
  String get usageUnknown => 'Лимиты неизвестны';

  @override
  String noActiveAccount(String service) {
    return 'Нет активного аккаунта $service';
  }

  @override
  String availableAgain(String time, String relative) {
    return 'Снова доступен: $time · $relative';
  }

  @override
  String usageChecked(String ago) {
    return 'Лимиты проверены $ago';
  }

  @override
  String get usageNotChecked => 'Лимиты ещё не проверены';

  @override
  String get usageStale => 'Последние известные данные · компьютер не в сети';

  @override
  String get forNewTasks => 'Для новых задач';

  @override
  String get useForTasks => 'Выбрать';

  @override
  String limitTitle(String provider) {
    return 'Лимит $provider исчерпан';
  }

  @override
  String limitAccount(String service, String account) {
    return 'Аккаунт $service: $account';
  }

  @override
  String get limitNoReset =>
      'Время сброса неизвестно. Смените аккаунт или выберите другой ИИ.';

  @override
  String get switchAccount => 'Сменить аккаунт';

  @override
  String switchTo(String provider) {
    return 'Перейти на $provider';
  }

  @override
  String get windowFiveHour => '5 часов';

  @override
  String get windowWeekly => 'Неделя';

  @override
  String windowWeeklyModel(String model) {
    return 'Неделя · $model';
  }

  @override
  String usedPercent(int percent) {
    return '$percent %';
  }

  @override
  String resetsAt(String time, String relative) {
    return 'Сброс: $time · $relative';
  }

  @override
  String get taskTitle => 'Задача';

  @override
  String taskFrom(String time) {
    return 'Задача от $time';
  }

  @override
  String get queued => 'В очереди';

  @override
  String get running => 'Выполняется';

  @override
  String get completed => 'Готово';

  @override
  String get failed => 'Ошибка';

  @override
  String get stopped => 'Остановлена';

  @override
  String get unknownState => 'Уточняется';

  @override
  String get executor => 'Исполнитель';

  @override
  String get model => 'Модель';

  @override
  String get defaultModel => 'По умолчанию';

  @override
  String chooseModel(String provider) {
    return 'Модель $provider';
  }

  @override
  String get effort => 'Глубина рассуждений';

  @override
  String get effortLow => 'Низкая';

  @override
  String get effortMedium => 'Средняя';

  @override
  String get effortHigh => 'Высокая';

  @override
  String get effortXhigh => 'Очень высокая';

  @override
  String get effortMax => 'Максимальная';

  @override
  String get codexDefault =>
      'Codex · модель по умолчанию. Выбор ИИ и модели появится после обновления сервера.';

  @override
  String workspaceHint(String workspace) {
    return 'Проект «$workspace» на компьютере. Доступ только на чтение — файлы не изменяются.';
  }

  @override
  String get promptHint => 'Опишите задачу…';

  @override
  String get promptRequired => 'Введите текст задачи.';

  @override
  String get promptTooLong => 'Слишком длинный текст: максимум 16 КБ.';

  @override
  String get send => 'Отправить';

  @override
  String get suggestions => 'Можно начать с';

  @override
  String get suggestion1 =>
      'Кратко объясни назначение проекта по README. Ничего не изменяй.';

  @override
  String get suggestion2 => 'Найди основные точки входа приложения и опиши их.';

  @override
  String get suggestion3 =>
      'Проверь README на устаревшие инструкции и перечисли их.';

  @override
  String get requestUnknown =>
      'Сервер мог принять запрос, но ответ не дошёл. Повторите тот же запрос — вторая задача не создастся.';

  @override
  String get discardPending => 'Проверено, отменить';

  @override
  String get requestExpired =>
      'Срок безопасного повтора истёк. Проверьте историю задач перед новой отправкой.';

  @override
  String get blockNoServerTitle => 'Нет связи с сервером';

  @override
  String get blockOfflineTitle => 'Компьютер не в сети';

  @override
  String get blockOffline =>
      'Отправка станет доступна, когда мост Portway на компьютере выйдет на связь.';

  @override
  String blockNotInstalledTitle(String provider) {
    return '$provider не установлен';
  }

  @override
  String get blockNotInstalled =>
      'Установите CLI на компьютер или выберите другой ИИ.';

  @override
  String blockNoAccountTitle(String provider) {
    return 'Нет аккаунта для $provider';
  }

  @override
  String blockNoAccount(String service) {
    return 'Подключите аккаунт $service, чтобы отправлять задачи.';
  }

  @override
  String get connectAccount => 'Подключить аккаунт';

  @override
  String get openTasks => 'Открыть задачи';

  @override
  String progressCount(int count) {
    return 'Ход выполнения · $count';
  }

  @override
  String get emptyEvents => 'Событий нет';

  @override
  String get emptyEventsActive =>
      'Компьютер принял задачу, события появятся здесь';

  @override
  String get eventStarted => 'Начато';

  @override
  String get eventProgress => 'Ход работы';

  @override
  String get eventCompleted => 'Завершено';

  @override
  String get eventFailed => 'Ошибка';

  @override
  String get eventStopped => 'Остановлено';

  @override
  String get waitingResult => 'Ответ появится здесь, как только ИИ закончит.';

  @override
  String answerFrom(String provider) {
    return 'Ответ · $provider';
  }

  @override
  String get stop => 'Остановить';

  @override
  String get stopTitle => 'Остановить задачу?';

  @override
  String get stopHint =>
      'Уже полученный ответ сохранится. Продолжить этот запуск будет нельзя.';

  @override
  String get stoppedTitle => 'Задача остановлена';

  @override
  String get stoppedHint =>
      'Остановка подтверждена сервером; процесс на компьютере завершится в течение нескольких секунд.';

  @override
  String get failedHint =>
      'Можно отправить задачу заново после устранения причины.';

  @override
  String get runLimit => 'Лимит ИИ исчерпан во время выполнения';

  @override
  String get runAuth => 'Нужно заново войти в аккаунт ИИ на компьютере';

  @override
  String get runBridgeLost =>
      'Связь с компьютером прервалась во время выполнения';

  @override
  String get runTimeout => 'Превышено время выполнения (9 минут)';

  @override
  String get runFailedGeneric => 'ИИ завершился с ошибкой';

  @override
  String get account => 'Аккаунт';

  @override
  String get tokens => 'Токены';

  @override
  String tokensValue(String input, String output) {
    return '$input на входе · $output на выходе';
  }

  @override
  String get duration => 'Длительность';

  @override
  String get created => 'Создана';

  @override
  String get workspace => 'Проект';

  @override
  String workspaceValue(String workspace) {
    return '$workspace · только чтение';
  }

  @override
  String get runId => 'ID задачи';

  @override
  String get searchHint => 'Поиск по задачам';

  @override
  String get clearSearch => 'Очистить поиск';

  @override
  String get filterAll => 'Все';

  @override
  String get filterActive => 'Активные';

  @override
  String get filterCompleted => 'Готовые';

  @override
  String get filterFailed => 'С ошибкой';

  @override
  String get filterStopped => 'Остановленные';

  @override
  String get allProviders => 'Все ИИ';

  @override
  String foundCount(int found, int total) {
    return 'Найдено $found из $total';
  }

  @override
  String totalCount(int count) {
    return 'Всего задач: $count';
  }

  @override
  String get resetFilters => 'Сбросить';

  @override
  String get nothingFound => 'Ничего не найдено';

  @override
  String get nothingFoundHint => 'Измените запрос или сбросьте фильтры.';

  @override
  String get aiAccountsHint =>
      'Аккаунты ChatGPT и Claude, через которые компьютер запускает Codex и Claude Code.';

  @override
  String get accountsOfflineTitle => 'Компьютер не в сети';

  @override
  String get accountsOfflineStale =>
      'Показаны последние известные данные. Управление аккаунтами недоступно до подключения.';

  @override
  String get installed => 'Установлен';

  @override
  String installedVersion(String version) {
    return 'CLI $version';
  }

  @override
  String providerService(String service) {
    return 'Аккаунт $service';
  }

  @override
  String cliMissing(String provider) {
    return 'Установите $provider на компьютер, чтобы подключить аккаунт.';
  }

  @override
  String noAccounts(String service) {
    return 'Аккаунты $service ещё не подключены.';
  }

  @override
  String get activeAccount => 'Активный';

  @override
  String get systemAccount => 'Системный';

  @override
  String get accountActions => 'Действия с аккаунтом';

  @override
  String get makeActive => 'Сделать активным';

  @override
  String get accountActivated => 'Аккаунт выбран активным';

  @override
  String get accountRemoved => 'Аккаунт удалён';

  @override
  String get removeAccountTitle => 'Удалить аккаунт?';

  @override
  String removeAccountHint(String account, String provider) {
    return 'Компьютер выйдет из аккаунта $account для $provider и удалит его профиль. История задач сохранится.';
  }

  @override
  String connectAccountTitle(String service) {
    return 'Вход в $service';
  }

  @override
  String get connectAccountHint =>
      'Где удобнее войти? Пароль вводится только на странице сервиса.';

  @override
  String get methodPhone => 'На этом телефоне';

  @override
  String get methodPhoneCodex =>
      'Покажем код и откроем страницу входа ChatGPT в браузере телефона.';

  @override
  String get methodPhoneClaude =>
      'Откроем страницу Claude в браузере телефона, затем вставьте полученный код.';

  @override
  String get methodComputer => 'На компьютере';

  @override
  String get methodComputerHint =>
      'Браузер откроется на компьютере, войдите там.';

  @override
  String get loginActiveTitle => 'Предыдущий вход не завершён';

  @override
  String get loginActiveHint => 'Отменить его и начать новый?';

  @override
  String get cancelPrevious => 'Отменить предыдущий';

  @override
  String loginInProgress(String provider) {
    return 'Идёт вход в $provider';
  }

  @override
  String get loginInProgressHint => 'Откройте, чтобы продолжить или отменить.';

  @override
  String get loginStarting => 'Компьютер готовит вход…';

  @override
  String get loginStartingHint => 'Обычно это занимает несколько секунд.';

  @override
  String get loginUrlPending => 'Ждём ссылку на страницу входа.';

  @override
  String get deviceCodeTitle => 'Код для входа';

  @override
  String get deviceCodeSteps =>
      '1. Скопируйте код.\n2. Откройте страницу входа и войдите в ChatGPT.\n3. Введите код и подтвердите доступ.';

  @override
  String get claudeSteps =>
      '1. Откройте страницу входа и войдите в Claude.\n2. Подтвердите доступ и скопируйте показанный код.\n3. Вставьте код ниже.';

  @override
  String get openLoginPage => 'Открыть страницу входа';

  @override
  String get openFailed => 'Не удалось открыть браузер';

  @override
  String get pasteCode => 'Вставьте код со страницы Claude';

  @override
  String get sendCode => 'Отправить код';

  @override
  String get codeInvalid => 'Вставьте код целиком, без пробелов.';

  @override
  String get loginWaiting => 'Ждём подтверждения';

  @override
  String get loginWaitingHint => 'Экран обновится автоматически после входа.';

  @override
  String get loginOnComputer => 'Войдите на компьютере';

  @override
  String loginOnComputerHint(String service) {
    return 'На компьютере открылся браузер. Войдите в аккаунт $service — этот экран обновится сам.';
  }

  @override
  String get loginVerifying => 'Проверяем вход…';

  @override
  String get loginVerifyingHint => 'Компьютер сохраняет аккаунт.';

  @override
  String get loginCompleted => 'Аккаунт подключён';

  @override
  String get loginCompletedHint => 'Он выбран активным для новых задач.';

  @override
  String get loginFailed => 'Вход не выполнен';

  @override
  String get loginExpired => 'Время на вход истекло';

  @override
  String get loginCancelled => 'Вход отменён';

  @override
  String get loginFailedHint =>
      'Попробуйте ещё раз или выберите вход на компьютере.';

  @override
  String loginExpires(String time) {
    return 'Вход нужно завершить до $time';
  }

  @override
  String get cancelLogin => 'Отменить вход';

  @override
  String get deviceCodeUnavailable =>
      'ChatGPT не разрешает вход по коду для этого аккаунта. Включите его в ChatGPT → Settings → Security или выберите «На компьютере».';

  @override
  String get codeRejected =>
      'Код не принят. Начните вход заново и вставьте новый код.';

  @override
  String get cliMissingShort => 'CLI не найден на компьютере.';

  @override
  String get loginTimeout => 'Вход не завершён вовремя. Начните заново.';

  @override
  String get ownerRole => 'Владелец установки';

  @override
  String get connection => 'Подключение';

  @override
  String get installation => 'Установка';

  @override
  String get deviceSessions => 'Устройства';

  @override
  String get noDevices => 'Нет данных об устройствах';

  @override
  String get unknownDevice => 'Устройство';

  @override
  String get thisDevice => 'Это устройство';

  @override
  String deviceSeen(String ago) {
    return 'Активность: $ago';
  }

  @override
  String get deviceNeverSeen => 'Активности ещё не было';

  @override
  String get deviceRevokedLabel => 'Доступ отозван';

  @override
  String get revoke => 'Отозвать доступ';

  @override
  String get revokeTitle => 'Отозвать доступ?';

  @override
  String revokeHint(String device) {
    return 'Устройство «$device» потеряет доступ и должно будет войти заново.';
  }

  @override
  String get theme => 'Оформление';

  @override
  String get systemTheme => 'Системная';

  @override
  String get lightTheme => 'Светлая';

  @override
  String get darkTheme => 'Тёмная';

  @override
  String get about => 'О приложении';

  @override
  String get appVersion => 'Версия';

  @override
  String get security => 'Безопасность';

  @override
  String get securityValue =>
      'Сессия в защищённом хранилище Android · ключи ИИ остаются на компьютере';

  @override
  String get logout => 'Выйти';

  @override
  String get logoutTitle => 'Выйти из аккаунта?';

  @override
  String get logoutHint =>
      'Сессия на этом телефоне будет завершена. Чтобы снова открыть разделы, потребуется войти.';

  @override
  String get logoutFootnote =>
      'Задачи и аккаунты ИИ на компьютере не удаляются.';

  @override
  String get ownerAlreadyBound =>
      'Владелец уже зарегистрирован. Войдите с его email и паролем.';

  @override
  String get ownerRequired =>
      'Этот аккаунт не совпадает с владельцем установки.';

  @override
  String get registrationUncertain =>
      'Исход регистрации не подтверждён. Проверьте письмо перед повтором.';

  @override
  String get registrationConflict =>
      'Регистрация уже начата. Подтвердите ранее указанный email и войдите.';

  @override
  String get rateLimited =>
      'Слишком много попыток. Подождите минуту и повторите.';

  @override
  String get invalidUrl =>
      'Нужен HTTPS-адрес без пути, параметров и localhost.';

  @override
  String get networkError =>
      'Не удалось связаться с сервером. Проверьте интернет и доступность компьютера.';

  @override
  String get responseError => 'Сервер вернул неподдерживаемый ответ.';

  @override
  String get storageError =>
      'Не удалось открыть защищённое хранилище. Перезапустите приложение.';

  @override
  String get notConfirmed => 'Сначала подтвердите email по ссылке из письма.';

  @override
  String get setupError => 'Неверный код настройки.';

  @override
  String get errorOffline =>
      'Компьютер не на связи. Запустите мост Portway на компьютере и повторите.';

  @override
  String get disabled => 'Подключение ИИ на сервере ещё не настроено.';

  @override
  String get busyError =>
      'Компьютер уже выполняет другую задачу. Дождитесь её завершения.';

  @override
  String get errorLimit => 'Лимит выбранного ИИ исчерпан.';

  @override
  String get errorAccountRequired =>
      'Для выбранного ИИ нет аккаунта с выполненным входом.';

  @override
  String get errorProviderUnavailable =>
      'Выбранный ИИ не установлен на компьютере.';

  @override
  String get errorLoginActive => 'Уже идёт другой вход в аккаунт.';

  @override
  String get errorAccountSystem =>
      'Системный аккаунт компьютера нельзя удалить из приложения.';

  @override
  String get errorInvalidRequest => 'Сервер не принял параметры запроса.';

  @override
  String get eventGap =>
      'Часть событий пропущена. Потяните вниз, чтобы обновить.';

  @override
  String get notFound => 'Не найдено. Возможно, сервер ещё не обновлён.';

  @override
  String get serverError => 'Не удалось выполнить запрос. Попробуйте ещё раз.';

  @override
  String get errorLoginNotActive =>
      'Этот вход уже завершён или отменён. Начните вход заново.';

  @override
  String get errorCommandsBusy =>
      'Компьютер обрабатывает много команд. Повторите через минуту.';

  @override
  String get commandExpired =>
      'Компьютер не забрал команду вовремя. Проверьте, что мост Portway запущен.';

  @override
  String get accountNotFound =>
      'Аккаунт не найден на компьютере. Обновите список.';

  @override
  String get accountBusy =>
      'Аккаунт занят выполняемой задачей. Дождитесь её завершения.';

  @override
  String get commandFailedGeneric =>
      'Компьютер не выполнил команду. Попробуйте ещё раз.';
}
