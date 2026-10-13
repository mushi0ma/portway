import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_ru.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations? of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations);
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[Locale('ru')];

  /// No description provided for @appTitle.
  ///
  /// In ru, this message translates to:
  /// **'Portway'**
  String get appTitle;

  /// No description provided for @connectTitle.
  ///
  /// In ru, this message translates to:
  /// **'Подключение к компьютеру'**
  String get connectTitle;

  /// No description provided for @connectHint.
  ///
  /// In ru, this message translates to:
  /// **'Укажите HTTPS-адрес Portway, запущенного на вашем компьютере.'**
  String get connectHint;

  /// No description provided for @connectWhereTitle.
  ///
  /// In ru, this message translates to:
  /// **'Где взять адрес?'**
  String get connectWhereTitle;

  /// No description provided for @connectWhere.
  ///
  /// In ru, this message translates to:
  /// **'Его показывает Portway на компьютере после запуска (Tailscale Funnel). Адрес localhost с телефона не работает.'**
  String get connectWhere;

  /// No description provided for @apiUrl.
  ///
  /// In ru, this message translates to:
  /// **'Адрес сервера'**
  String get apiUrl;

  /// No description provided for @urlHint.
  ///
  /// In ru, this message translates to:
  /// **'https://portway.example.ts.net'**
  String get urlHint;

  /// No description provided for @urlHelper.
  ///
  /// In ru, this message translates to:
  /// **'Только HTTPS, без пути и параметров'**
  String get urlHelper;

  /// No description provided for @urlRequired.
  ///
  /// In ru, this message translates to:
  /// **'Введите адрес сервера'**
  String get urlRequired;

  /// No description provided for @connect.
  ///
  /// In ru, this message translates to:
  /// **'Подключить'**
  String get connect;

  /// No description provided for @loginTitle.
  ///
  /// In ru, this message translates to:
  /// **'Вход'**
  String get loginTitle;

  /// No description provided for @loginAt.
  ///
  /// In ru, this message translates to:
  /// **'Войдите в «{installation}», чтобы управлять ИИ на компьютере.'**
  String loginAt(String installation);

  /// No description provided for @email.
  ///
  /// In ru, this message translates to:
  /// **'Email'**
  String get email;

  /// No description provided for @password.
  ///
  /// In ru, this message translates to:
  /// **'Пароль'**
  String get password;

  /// No description provided for @name.
  ///
  /// In ru, this message translates to:
  /// **'Имя'**
  String get name;

  /// No description provided for @setupProof.
  ///
  /// In ru, this message translates to:
  /// **'Код настройки'**
  String get setupProof;

  /// No description provided for @setupProofHelper.
  ///
  /// In ru, this message translates to:
  /// **'Значение PORTWAY_SETUP_TOKEN из файла .env на компьютере'**
  String get setupProofHelper;

  /// No description provided for @setupProofRequired.
  ///
  /// In ru, this message translates to:
  /// **'Введите код настройки'**
  String get setupProofRequired;

  /// No description provided for @login.
  ///
  /// In ru, this message translates to:
  /// **'Войти'**
  String get login;

  /// No description provided for @noAccount.
  ///
  /// In ru, this message translates to:
  /// **'Нет аккаунта?'**
  String get noAccount;

  /// No description provided for @haveAccount.
  ///
  /// In ru, this message translates to:
  /// **'Уже есть аккаунт?'**
  String get haveAccount;

  /// No description provided for @createAccount.
  ///
  /// In ru, this message translates to:
  /// **'Зарегистрироваться'**
  String get createAccount;

  /// No description provided for @register.
  ///
  /// In ru, this message translates to:
  /// **'Зарегистрироваться'**
  String get register;

  /// No description provided for @registrationTitle.
  ///
  /// In ru, this message translates to:
  /// **'Регистрация владельца'**
  String get registrationTitle;

  /// No description provided for @registrationHint.
  ///
  /// In ru, this message translates to:
  /// **'Владелец установки «{installation}» — её единственный пользователь. После регистрации подтвердите email.'**
  String registrationHint(String installation);

  /// No description provided for @repeatPassword.
  ///
  /// In ru, this message translates to:
  /// **'Повторите пароль'**
  String get repeatPassword;

  /// No description provided for @repeatRequired.
  ///
  /// In ru, this message translates to:
  /// **'Введите пароль ещё раз'**
  String get repeatRequired;

  /// No description provided for @passwordMismatch.
  ///
  /// In ru, this message translates to:
  /// **'Пароли не совпадают'**
  String get passwordMismatch;

  /// No description provided for @ruleLength.
  ///
  /// In ru, this message translates to:
  /// **'Не менее 8 символов'**
  String get ruleLength;

  /// No description provided for @ruleMix.
  ///
  /// In ru, this message translates to:
  /// **'Есть буква и цифра'**
  String get ruleMix;

  /// No description provided for @nameRequired.
  ///
  /// In ru, this message translates to:
  /// **'Введите имя'**
  String get nameRequired;

  /// No description provided for @nameTooLong.
  ///
  /// In ru, this message translates to:
  /// **'Не более 100 символов'**
  String get nameTooLong;

  /// No description provided for @emailRequired.
  ///
  /// In ru, this message translates to:
  /// **'Введите email'**
  String get emailRequired;

  /// No description provided for @invalidEmail.
  ///
  /// In ru, this message translates to:
  /// **'Некорректный email. Пример: name@example.com'**
  String get invalidEmail;

  /// No description provided for @passwordRequired.
  ///
  /// In ru, this message translates to:
  /// **'Введите пароль'**
  String get passwordRequired;

  /// No description provided for @shortPassword.
  ///
  /// In ru, this message translates to:
  /// **'Пароль должен содержать не менее 8 символов'**
  String get shortPassword;

  /// No description provided for @weakPassword.
  ///
  /// In ru, this message translates to:
  /// **'Добавьте хотя бы одну букву и одну цифру'**
  String get weakPassword;

  /// No description provided for @confirmationTitle.
  ///
  /// In ru, this message translates to:
  /// **'Подтвердите email'**
  String get confirmationTitle;

  /// No description provided for @confirmationHint.
  ///
  /// In ru, this message translates to:
  /// **'Мы отправили письмо на {email}. Перейдите по ссылке из письма, затем войдите с тем же паролем.'**
  String confirmationHint(String email);

  /// No description provided for @confirmationStepsTitle.
  ///
  /// In ru, this message translates to:
  /// **'Ссылка откроет приложение'**
  String get confirmationStepsTitle;

  /// No description provided for @confirmationSteps.
  ///
  /// In ru, this message translates to:
  /// **'Если письмо открыто на этом телефоне, Portway сам вернётся на экран входа. Проверьте папку «Спам», если письма нет.'**
  String get confirmationSteps;

  /// No description provided for @editRegistration.
  ///
  /// In ru, this message translates to:
  /// **'Изменить данные'**
  String get editRegistration;

  /// No description provided for @returnLogin.
  ///
  /// In ru, this message translates to:
  /// **'Перейти ко входу'**
  String get returnLogin;

  /// No description provided for @change.
  ///
  /// In ru, this message translates to:
  /// **'Сменить'**
  String get change;

  /// No description provided for @changeInstallation.
  ///
  /// In ru, this message translates to:
  /// **'Сменить установку'**
  String get changeInstallation;

  /// No description provided for @changeInstallationTitle.
  ///
  /// In ru, this message translates to:
  /// **'Сменить установку?'**
  String get changeInstallationTitle;

  /// No description provided for @changeConfirm.
  ///
  /// In ru, this message translates to:
  /// **'Сессия на этом устройстве будет завершена. Для другой установки потребуется войти заново.'**
  String get changeConfirm;

  /// No description provided for @changeConfirmSignedOut.
  ///
  /// In ru, this message translates to:
  /// **'Приложение забудет адрес этой установки. Затем укажите адрес другого компьютера.'**
  String get changeConfirmSignedOut;

  /// No description provided for @cancel.
  ///
  /// In ru, this message translates to:
  /// **'Отмена'**
  String get cancel;

  /// No description provided for @continueAction.
  ///
  /// In ru, this message translates to:
  /// **'Продолжить'**
  String get continueAction;

  /// No description provided for @showPassword.
  ///
  /// In ru, this message translates to:
  /// **'Показать пароль'**
  String get showPassword;

  /// No description provided for @hidePassword.
  ///
  /// In ru, this message translates to:
  /// **'Скрыть пароль'**
  String get hidePassword;

  /// No description provided for @credentialsTitle.
  ///
  /// In ru, this message translates to:
  /// **'Не удалось войти'**
  String get credentialsTitle;

  /// No description provided for @credentialsError.
  ///
  /// In ru, this message translates to:
  /// **'Неверный email или пароль. Проверьте данные и попробуйте снова.'**
  String get credentialsError;

  /// No description provided for @sessionExpiredTitle.
  ///
  /// In ru, this message translates to:
  /// **'Сессия завершена'**
  String get sessionExpiredTitle;

  /// No description provided for @sessionExpired.
  ///
  /// In ru, this message translates to:
  /// **'Срок входа истёк. Войдите снова.'**
  String get sessionExpired;

  /// No description provided for @deviceRevokedTitle.
  ///
  /// In ru, this message translates to:
  /// **'Доступ устройства отозван'**
  String get deviceRevokedTitle;

  /// No description provided for @deviceRevoked.
  ///
  /// In ru, this message translates to:
  /// **'Это устройство отключено от установки. Войдите снова.'**
  String get deviceRevoked;

  /// No description provided for @signedOut.
  ///
  /// In ru, this message translates to:
  /// **'Вы вышли из аккаунта'**
  String get signedOut;

  /// No description provided for @signedOutHint.
  ///
  /// In ru, this message translates to:
  /// **'Разделы приложения снова доступны только после входа.'**
  String get signedOutHint;

  /// No description provided for @logoutPending.
  ///
  /// In ru, this message translates to:
  /// **'Выход выполнен на телефоне; сервер подтвердит отзыв доступа при следующем подключении.'**
  String get logoutPending;

  /// No description provided for @registrationUnknownTitle.
  ///
  /// In ru, this message translates to:
  /// **'Ответ на регистрацию не получен'**
  String get registrationUnknownTitle;

  /// No description provided for @registrationUnknown.
  ///
  /// In ru, this message translates to:
  /// **'Проверьте почту. Повтор отправит тот же запрос и не создаст второй аккаунт.'**
  String get registrationUnknown;

  /// No description provided for @retrySame.
  ///
  /// In ru, this message translates to:
  /// **'Повторить тот же запрос'**
  String get retrySame;

  /// No description provided for @home.
  ///
  /// In ru, this message translates to:
  /// **'Главная'**
  String get home;

  /// No description provided for @tasks.
  ///
  /// In ru, this message translates to:
  /// **'Задачи'**
  String get tasks;

  /// No description provided for @aiAccounts.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунты ИИ'**
  String get aiAccounts;

  /// No description provided for @profile.
  ///
  /// In ru, this message translates to:
  /// **'Профиль'**
  String get profile;

  /// No description provided for @loading.
  ///
  /// In ru, this message translates to:
  /// **'Загрузка'**
  String get loading;

  /// No description provided for @retry.
  ///
  /// In ru, this message translates to:
  /// **'Повторить'**
  String get retry;

  /// No description provided for @refresh.
  ///
  /// In ru, this message translates to:
  /// **'Обновить'**
  String get refresh;

  /// No description provided for @copy.
  ///
  /// In ru, this message translates to:
  /// **'Копировать'**
  String get copy;

  /// No description provided for @copied.
  ///
  /// In ru, this message translates to:
  /// **'Скопировано'**
  String get copied;

  /// No description provided for @open.
  ///
  /// In ru, this message translates to:
  /// **'Открыть'**
  String get open;

  /// No description provided for @done.
  ///
  /// In ru, this message translates to:
  /// **'Готово'**
  String get done;

  /// No description provided for @remove.
  ///
  /// In ru, this message translates to:
  /// **'Удалить'**
  String get remove;

  /// No description provided for @manage.
  ///
  /// In ru, this message translates to:
  /// **'Управление'**
  String get manage;

  /// No description provided for @details.
  ///
  /// In ru, this message translates to:
  /// **'Подробности'**
  String get details;

  /// No description provided for @code.
  ///
  /// In ru, this message translates to:
  /// **'код'**
  String get code;

  /// No description provided for @copyCode.
  ///
  /// In ru, this message translates to:
  /// **'Скопировать код'**
  String get copyCode;

  /// No description provided for @copyAnswer.
  ///
  /// In ru, this message translates to:
  /// **'Скопировать ответ'**
  String get copyAnswer;

  /// No description provided for @greeting.
  ///
  /// In ru, this message translates to:
  /// **'{dayPart}, {name}'**
  String greeting(String dayPart, String name);

  /// No description provided for @morning.
  ///
  /// In ru, this message translates to:
  /// **'Доброе утро'**
  String get morning;

  /// No description provided for @day.
  ///
  /// In ru, this message translates to:
  /// **'Добрый день'**
  String get day;

  /// No description provided for @evening.
  ///
  /// In ru, this message translates to:
  /// **'Добрый вечер'**
  String get evening;

  /// No description provided for @night.
  ///
  /// In ru, this message translates to:
  /// **'Доброй ночи'**
  String get night;

  /// No description provided for @ownerFallback.
  ///
  /// In ru, this message translates to:
  /// **'владелец'**
  String get ownerFallback;

  /// No description provided for @computer.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер'**
  String get computer;

  /// No description provided for @online.
  ///
  /// In ru, this message translates to:
  /// **'На связи'**
  String get online;

  /// No description provided for @offline.
  ///
  /// In ru, this message translates to:
  /// **'Не в сети'**
  String get offline;

  /// No description provided for @statusUnknown.
  ///
  /// In ru, this message translates to:
  /// **'Связь проверяется'**
  String get statusUnknown;

  /// No description provided for @statusChecking.
  ///
  /// In ru, this message translates to:
  /// **'Запрашиваем состояние компьютера…'**
  String get statusChecking;

  /// No description provided for @computerOnlineHint.
  ///
  /// In ru, this message translates to:
  /// **'Проект «{workspace}» · только чтение'**
  String computerOnlineHint(String workspace);

  /// No description provided for @computerLastSeen.
  ///
  /// In ru, this message translates to:
  /// **'Был на связи {ago}'**
  String computerLastSeen(String ago);

  /// No description provided for @computerNeverSeen.
  ///
  /// In ru, this message translates to:
  /// **'Мост Portway на компьютере ещё не подключался'**
  String get computerNeverSeen;

  /// No description provided for @offlineHint.
  ///
  /// In ru, this message translates to:
  /// **'Запустите мост Portway на компьютере — статус обновится автоматически.'**
  String get offlineHint;

  /// No description provided for @newTask.
  ///
  /// In ru, this message translates to:
  /// **'Новая задача'**
  String get newTask;

  /// No description provided for @pendingTitle.
  ///
  /// In ru, this message translates to:
  /// **'Проверить отправку'**
  String get pendingTitle;

  /// No description provided for @aiModels.
  ///
  /// In ru, this message translates to:
  /// **'Модели ИИ'**
  String get aiModels;

  /// No description provided for @usageLoading.
  ///
  /// In ru, this message translates to:
  /// **'Загружаем аккаунты и лимиты…'**
  String get usageLoading;

  /// No description provided for @providersPending.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер передаст данные об аккаунтах и лимитах после обновления сервера и моста.'**
  String get providersPending;

  /// No description provided for @providersOffline.
  ///
  /// In ru, this message translates to:
  /// **'Данные об аккаунтах ИИ появятся, когда компьютер выйдет на связь.'**
  String get providersOffline;

  /// No description provided for @providersEmptyTitle.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунтов пока нет'**
  String get providersEmptyTitle;

  /// No description provided for @recentTasks.
  ///
  /// In ru, this message translates to:
  /// **'Последние задачи'**
  String get recentTasks;

  /// No description provided for @allTasks.
  ///
  /// In ru, this message translates to:
  /// **'Все задачи'**
  String get allTasks;

  /// No description provided for @noTasks.
  ///
  /// In ru, this message translates to:
  /// **'Задач пока нет'**
  String get noTasks;

  /// No description provided for @noTasksHint.
  ///
  /// In ru, this message translates to:
  /// **'Начните с вопроса о проекте — например, попросите объяснить его структуру.'**
  String get noTasksHint;

  /// No description provided for @available.
  ///
  /// In ru, this message translates to:
  /// **'Доступен'**
  String get available;

  /// No description provided for @notInstalled.
  ///
  /// In ru, this message translates to:
  /// **'Не установлен'**
  String get notInstalled;

  /// No description provided for @noLogin.
  ///
  /// In ru, this message translates to:
  /// **'Нет входа'**
  String get noLogin;

  /// No description provided for @limitExhausted.
  ///
  /// In ru, this message translates to:
  /// **'Лимит исчерпан'**
  String get limitExhausted;

  /// No description provided for @limitNear.
  ///
  /// In ru, this message translates to:
  /// **'Лимит почти исчерпан'**
  String get limitNear;

  /// No description provided for @usageUnknown.
  ///
  /// In ru, this message translates to:
  /// **'Лимиты неизвестны'**
  String get usageUnknown;

  /// No description provided for @noActiveAccount.
  ///
  /// In ru, this message translates to:
  /// **'Нет активного аккаунта {service}'**
  String noActiveAccount(String service);

  /// No description provided for @availableAgain.
  ///
  /// In ru, this message translates to:
  /// **'Снова доступен: {time} · {relative}'**
  String availableAgain(String time, String relative);

  /// No description provided for @usageChecked.
  ///
  /// In ru, this message translates to:
  /// **'Лимиты проверены {ago}'**
  String usageChecked(String ago);

  /// No description provided for @usageNotChecked.
  ///
  /// In ru, this message translates to:
  /// **'Лимиты ещё не проверены'**
  String get usageNotChecked;

  /// No description provided for @usageStale.
  ///
  /// In ru, this message translates to:
  /// **'Последние известные данные · компьютер не в сети'**
  String get usageStale;

  /// No description provided for @forNewTasks.
  ///
  /// In ru, this message translates to:
  /// **'Для новых задач'**
  String get forNewTasks;

  /// No description provided for @useForTasks.
  ///
  /// In ru, this message translates to:
  /// **'Выбрать'**
  String get useForTasks;

  /// No description provided for @limitTitle.
  ///
  /// In ru, this message translates to:
  /// **'Лимит {provider} исчерпан'**
  String limitTitle(String provider);

  /// No description provided for @limitAccount.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунт {service}: {account}'**
  String limitAccount(String service, String account);

  /// No description provided for @limitNoReset.
  ///
  /// In ru, this message translates to:
  /// **'Время сброса неизвестно. Смените аккаунт или выберите другой ИИ.'**
  String get limitNoReset;

  /// No description provided for @switchAccount.
  ///
  /// In ru, this message translates to:
  /// **'Сменить аккаунт'**
  String get switchAccount;

  /// No description provided for @switchTo.
  ///
  /// In ru, this message translates to:
  /// **'Перейти на {provider}'**
  String switchTo(String provider);

  /// No description provided for @windowFiveHour.
  ///
  /// In ru, this message translates to:
  /// **'5 часов'**
  String get windowFiveHour;

  /// No description provided for @windowWeekly.
  ///
  /// In ru, this message translates to:
  /// **'Неделя'**
  String get windowWeekly;

  /// No description provided for @windowWeeklyModel.
  ///
  /// In ru, this message translates to:
  /// **'Неделя · {model}'**
  String windowWeeklyModel(String model);

  /// No description provided for @usedPercent.
  ///
  /// In ru, this message translates to:
  /// **'{percent} %'**
  String usedPercent(int percent);

  /// No description provided for @resetsAt.
  ///
  /// In ru, this message translates to:
  /// **'Сброс: {time} · {relative}'**
  String resetsAt(String time, String relative);

  /// No description provided for @taskTitle.
  ///
  /// In ru, this message translates to:
  /// **'Задача'**
  String get taskTitle;

  /// No description provided for @taskFrom.
  ///
  /// In ru, this message translates to:
  /// **'Задача от {time}'**
  String taskFrom(String time);

  /// No description provided for @queued.
  ///
  /// In ru, this message translates to:
  /// **'В очереди'**
  String get queued;

  /// No description provided for @running.
  ///
  /// In ru, this message translates to:
  /// **'Выполняется'**
  String get running;

  /// No description provided for @completed.
  ///
  /// In ru, this message translates to:
  /// **'Готово'**
  String get completed;

  /// No description provided for @failed.
  ///
  /// In ru, this message translates to:
  /// **'Ошибка'**
  String get failed;

  /// No description provided for @stopped.
  ///
  /// In ru, this message translates to:
  /// **'Остановлена'**
  String get stopped;

  /// No description provided for @unknownState.
  ///
  /// In ru, this message translates to:
  /// **'Уточняется'**
  String get unknownState;

  /// No description provided for @executor.
  ///
  /// In ru, this message translates to:
  /// **'Исполнитель'**
  String get executor;

  /// No description provided for @model.
  ///
  /// In ru, this message translates to:
  /// **'Модель'**
  String get model;

  /// No description provided for @defaultModel.
  ///
  /// In ru, this message translates to:
  /// **'По умолчанию'**
  String get defaultModel;

  /// No description provided for @chooseModel.
  ///
  /// In ru, this message translates to:
  /// **'Модель {provider}'**
  String chooseModel(String provider);

  /// No description provided for @effort.
  ///
  /// In ru, this message translates to:
  /// **'Глубина рассуждений'**
  String get effort;

  /// No description provided for @effortLow.
  ///
  /// In ru, this message translates to:
  /// **'Низкая'**
  String get effortLow;

  /// No description provided for @effortMedium.
  ///
  /// In ru, this message translates to:
  /// **'Средняя'**
  String get effortMedium;

  /// No description provided for @effortHigh.
  ///
  /// In ru, this message translates to:
  /// **'Высокая'**
  String get effortHigh;

  /// No description provided for @effortXhigh.
  ///
  /// In ru, this message translates to:
  /// **'Очень высокая'**
  String get effortXhigh;

  /// No description provided for @effortMax.
  ///
  /// In ru, this message translates to:
  /// **'Максимальная'**
  String get effortMax;

  /// No description provided for @codexDefault.
  ///
  /// In ru, this message translates to:
  /// **'Codex · модель по умолчанию. Выбор ИИ и модели появится после обновления сервера.'**
  String get codexDefault;

  /// No description provided for @workspaceHint.
  ///
  /// In ru, this message translates to:
  /// **'Проект «{workspace}» на компьютере. Доступ только на чтение — файлы не изменяются.'**
  String workspaceHint(String workspace);

  /// No description provided for @promptHint.
  ///
  /// In ru, this message translates to:
  /// **'Опишите задачу…'**
  String get promptHint;

  /// No description provided for @promptRequired.
  ///
  /// In ru, this message translates to:
  /// **'Введите текст задачи.'**
  String get promptRequired;

  /// No description provided for @promptTooLong.
  ///
  /// In ru, this message translates to:
  /// **'Слишком длинный текст: максимум 16 КБ.'**
  String get promptTooLong;

  /// No description provided for @send.
  ///
  /// In ru, this message translates to:
  /// **'Отправить'**
  String get send;

  /// No description provided for @suggestions.
  ///
  /// In ru, this message translates to:
  /// **'Можно начать с'**
  String get suggestions;

  /// No description provided for @suggestion1.
  ///
  /// In ru, this message translates to:
  /// **'Кратко объясни назначение проекта по README. Ничего не изменяй.'**
  String get suggestion1;

  /// No description provided for @suggestion2.
  ///
  /// In ru, this message translates to:
  /// **'Найди основные точки входа приложения и опиши их.'**
  String get suggestion2;

  /// No description provided for @suggestion3.
  ///
  /// In ru, this message translates to:
  /// **'Проверь README на устаревшие инструкции и перечисли их.'**
  String get suggestion3;

  /// No description provided for @requestUnknown.
  ///
  /// In ru, this message translates to:
  /// **'Сервер мог принять запрос, но ответ не дошёл. Повторите тот же запрос — вторая задача не создастся.'**
  String get requestUnknown;

  /// No description provided for @discardPending.
  ///
  /// In ru, this message translates to:
  /// **'Проверено, отменить'**
  String get discardPending;

  /// No description provided for @requestExpired.
  ///
  /// In ru, this message translates to:
  /// **'Срок безопасного повтора истёк. Проверьте историю задач перед новой отправкой.'**
  String get requestExpired;

  /// No description provided for @blockNoServerTitle.
  ///
  /// In ru, this message translates to:
  /// **'Нет связи с сервером'**
  String get blockNoServerTitle;

  /// No description provided for @blockOfflineTitle.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер не в сети'**
  String get blockOfflineTitle;

  /// No description provided for @blockOffline.
  ///
  /// In ru, this message translates to:
  /// **'Отправка станет доступна, когда мост Portway на компьютере выйдет на связь.'**
  String get blockOffline;

  /// No description provided for @blockNotInstalledTitle.
  ///
  /// In ru, this message translates to:
  /// **'{provider} не установлен'**
  String blockNotInstalledTitle(String provider);

  /// No description provided for @blockNotInstalled.
  ///
  /// In ru, this message translates to:
  /// **'Установите CLI на компьютер или выберите другой ИИ.'**
  String get blockNotInstalled;

  /// No description provided for @blockNoAccountTitle.
  ///
  /// In ru, this message translates to:
  /// **'Нет аккаунта для {provider}'**
  String blockNoAccountTitle(String provider);

  /// No description provided for @blockNoAccount.
  ///
  /// In ru, this message translates to:
  /// **'Подключите аккаунт {service}, чтобы отправлять задачи.'**
  String blockNoAccount(String service);

  /// No description provided for @connectAccount.
  ///
  /// In ru, this message translates to:
  /// **'Подключить аккаунт'**
  String get connectAccount;

  /// No description provided for @openTasks.
  ///
  /// In ru, this message translates to:
  /// **'Открыть задачи'**
  String get openTasks;

  /// No description provided for @progressCount.
  ///
  /// In ru, this message translates to:
  /// **'Ход выполнения · {count}'**
  String progressCount(int count);

  /// No description provided for @emptyEvents.
  ///
  /// In ru, this message translates to:
  /// **'Событий нет'**
  String get emptyEvents;

  /// No description provided for @emptyEventsActive.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер принял задачу, события появятся здесь'**
  String get emptyEventsActive;

  /// No description provided for @eventStarted.
  ///
  /// In ru, this message translates to:
  /// **'Начато'**
  String get eventStarted;

  /// No description provided for @eventProgress.
  ///
  /// In ru, this message translates to:
  /// **'Ход работы'**
  String get eventProgress;

  /// No description provided for @eventCompleted.
  ///
  /// In ru, this message translates to:
  /// **'Завершено'**
  String get eventCompleted;

  /// No description provided for @eventFailed.
  ///
  /// In ru, this message translates to:
  /// **'Ошибка'**
  String get eventFailed;

  /// No description provided for @eventStopped.
  ///
  /// In ru, this message translates to:
  /// **'Остановлено'**
  String get eventStopped;

  /// No description provided for @waitingResult.
  ///
  /// In ru, this message translates to:
  /// **'Ответ появится здесь, как только ИИ закончит.'**
  String get waitingResult;

  /// No description provided for @answerFrom.
  ///
  /// In ru, this message translates to:
  /// **'Ответ · {provider}'**
  String answerFrom(String provider);

  /// No description provided for @stop.
  ///
  /// In ru, this message translates to:
  /// **'Остановить'**
  String get stop;

  /// No description provided for @stopTitle.
  ///
  /// In ru, this message translates to:
  /// **'Остановить задачу?'**
  String get stopTitle;

  /// No description provided for @stopHint.
  ///
  /// In ru, this message translates to:
  /// **'Уже полученный ответ сохранится. Продолжить этот запуск будет нельзя.'**
  String get stopHint;

  /// No description provided for @stoppedTitle.
  ///
  /// In ru, this message translates to:
  /// **'Задача остановлена'**
  String get stoppedTitle;

  /// No description provided for @stoppedHint.
  ///
  /// In ru, this message translates to:
  /// **'Остановка подтверждена сервером; процесс на компьютере завершится в течение нескольких секунд.'**
  String get stoppedHint;

  /// No description provided for @failedHint.
  ///
  /// In ru, this message translates to:
  /// **'Можно отправить задачу заново после устранения причины.'**
  String get failedHint;

  /// No description provided for @runLimit.
  ///
  /// In ru, this message translates to:
  /// **'Лимит ИИ исчерпан во время выполнения'**
  String get runLimit;

  /// No description provided for @runAuth.
  ///
  /// In ru, this message translates to:
  /// **'Нужно заново войти в аккаунт ИИ на компьютере'**
  String get runAuth;

  /// No description provided for @runBridgeLost.
  ///
  /// In ru, this message translates to:
  /// **'Связь с компьютером прервалась во время выполнения'**
  String get runBridgeLost;

  /// No description provided for @runTimeout.
  ///
  /// In ru, this message translates to:
  /// **'Превышено время выполнения (9 минут)'**
  String get runTimeout;

  /// No description provided for @runFailedGeneric.
  ///
  /// In ru, this message translates to:
  /// **'ИИ завершился с ошибкой'**
  String get runFailedGeneric;

  /// No description provided for @account.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунт'**
  String get account;

  /// No description provided for @tokens.
  ///
  /// In ru, this message translates to:
  /// **'Токены'**
  String get tokens;

  /// No description provided for @tokensValue.
  ///
  /// In ru, this message translates to:
  /// **'{input} на входе · {output} на выходе'**
  String tokensValue(String input, String output);

  /// No description provided for @duration.
  ///
  /// In ru, this message translates to:
  /// **'Длительность'**
  String get duration;

  /// No description provided for @created.
  ///
  /// In ru, this message translates to:
  /// **'Создана'**
  String get created;

  /// No description provided for @workspace.
  ///
  /// In ru, this message translates to:
  /// **'Проект'**
  String get workspace;

  /// No description provided for @workspaceValue.
  ///
  /// In ru, this message translates to:
  /// **'{workspace} · только чтение'**
  String workspaceValue(String workspace);

  /// No description provided for @runId.
  ///
  /// In ru, this message translates to:
  /// **'ID задачи'**
  String get runId;

  /// No description provided for @searchHint.
  ///
  /// In ru, this message translates to:
  /// **'Поиск по задачам'**
  String get searchHint;

  /// No description provided for @clearSearch.
  ///
  /// In ru, this message translates to:
  /// **'Очистить поиск'**
  String get clearSearch;

  /// No description provided for @filterAll.
  ///
  /// In ru, this message translates to:
  /// **'Все'**
  String get filterAll;

  /// No description provided for @filterActive.
  ///
  /// In ru, this message translates to:
  /// **'Активные'**
  String get filterActive;

  /// No description provided for @filterCompleted.
  ///
  /// In ru, this message translates to:
  /// **'Готовые'**
  String get filterCompleted;

  /// No description provided for @filterFailed.
  ///
  /// In ru, this message translates to:
  /// **'С ошибкой'**
  String get filterFailed;

  /// No description provided for @filterStopped.
  ///
  /// In ru, this message translates to:
  /// **'Остановленные'**
  String get filterStopped;

  /// No description provided for @allProviders.
  ///
  /// In ru, this message translates to:
  /// **'Все ИИ'**
  String get allProviders;

  /// No description provided for @foundCount.
  ///
  /// In ru, this message translates to:
  /// **'Найдено {found} из {total}'**
  String foundCount(int found, int total);

  /// No description provided for @totalCount.
  ///
  /// In ru, this message translates to:
  /// **'Всего задач: {count}'**
  String totalCount(int count);

  /// No description provided for @resetFilters.
  ///
  /// In ru, this message translates to:
  /// **'Сбросить'**
  String get resetFilters;

  /// No description provided for @nothingFound.
  ///
  /// In ru, this message translates to:
  /// **'Ничего не найдено'**
  String get nothingFound;

  /// No description provided for @nothingFoundHint.
  ///
  /// In ru, this message translates to:
  /// **'Измените запрос или сбросьте фильтры.'**
  String get nothingFoundHint;

  /// No description provided for @aiAccountsHint.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунты ChatGPT и Claude, через которые компьютер запускает Codex и Claude Code.'**
  String get aiAccountsHint;

  /// No description provided for @accountsOfflineTitle.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер не в сети'**
  String get accountsOfflineTitle;

  /// No description provided for @accountsOfflineStale.
  ///
  /// In ru, this message translates to:
  /// **'Показаны последние известные данные. Управление аккаунтами недоступно до подключения.'**
  String get accountsOfflineStale;

  /// No description provided for @installed.
  ///
  /// In ru, this message translates to:
  /// **'Установлен'**
  String get installed;

  /// No description provided for @installedVersion.
  ///
  /// In ru, this message translates to:
  /// **'CLI {version}'**
  String installedVersion(String version);

  /// No description provided for @providerService.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунт {service}'**
  String providerService(String service);

  /// No description provided for @cliMissing.
  ///
  /// In ru, this message translates to:
  /// **'Установите {provider} на компьютер, чтобы подключить аккаунт.'**
  String cliMissing(String provider);

  /// No description provided for @noAccounts.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунты {service} ещё не подключены.'**
  String noAccounts(String service);

  /// No description provided for @activeAccount.
  ///
  /// In ru, this message translates to:
  /// **'Активный'**
  String get activeAccount;

  /// No description provided for @systemAccount.
  ///
  /// In ru, this message translates to:
  /// **'Системный'**
  String get systemAccount;

  /// No description provided for @accountActions.
  ///
  /// In ru, this message translates to:
  /// **'Действия с аккаунтом'**
  String get accountActions;

  /// No description provided for @makeActive.
  ///
  /// In ru, this message translates to:
  /// **'Сделать активным'**
  String get makeActive;

  /// No description provided for @accountActivated.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунт выбран активным'**
  String get accountActivated;

  /// No description provided for @accountRemoved.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунт удалён'**
  String get accountRemoved;

  /// No description provided for @removeAccountTitle.
  ///
  /// In ru, this message translates to:
  /// **'Удалить аккаунт?'**
  String get removeAccountTitle;

  /// No description provided for @removeAccountHint.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер выйдет из аккаунта {account} для {provider} и удалит его профиль. История задач сохранится.'**
  String removeAccountHint(String account, String provider);

  /// No description provided for @connectAccountTitle.
  ///
  /// In ru, this message translates to:
  /// **'Вход в {service}'**
  String connectAccountTitle(String service);

  /// No description provided for @connectAccountHint.
  ///
  /// In ru, this message translates to:
  /// **'Где удобнее войти? Пароль вводится только на странице сервиса.'**
  String get connectAccountHint;

  /// No description provided for @methodPhone.
  ///
  /// In ru, this message translates to:
  /// **'На этом телефоне'**
  String get methodPhone;

  /// No description provided for @methodPhoneCodex.
  ///
  /// In ru, this message translates to:
  /// **'Покажем код и откроем страницу входа ChatGPT в браузере телефона.'**
  String get methodPhoneCodex;

  /// No description provided for @methodPhoneClaude.
  ///
  /// In ru, this message translates to:
  /// **'Откроем страницу Claude в браузере телефона, затем вставьте полученный код.'**
  String get methodPhoneClaude;

  /// No description provided for @methodComputer.
  ///
  /// In ru, this message translates to:
  /// **'На компьютере'**
  String get methodComputer;

  /// No description provided for @methodComputerHint.
  ///
  /// In ru, this message translates to:
  /// **'Браузер откроется на компьютере, войдите там.'**
  String get methodComputerHint;

  /// No description provided for @loginActiveTitle.
  ///
  /// In ru, this message translates to:
  /// **'Предыдущий вход не завершён'**
  String get loginActiveTitle;

  /// No description provided for @loginActiveHint.
  ///
  /// In ru, this message translates to:
  /// **'Отменить его и начать новый?'**
  String get loginActiveHint;

  /// No description provided for @cancelPrevious.
  ///
  /// In ru, this message translates to:
  /// **'Отменить предыдущий'**
  String get cancelPrevious;

  /// No description provided for @loginInProgress.
  ///
  /// In ru, this message translates to:
  /// **'Идёт вход в {provider}'**
  String loginInProgress(String provider);

  /// No description provided for @loginInProgressHint.
  ///
  /// In ru, this message translates to:
  /// **'Откройте, чтобы продолжить или отменить.'**
  String get loginInProgressHint;

  /// No description provided for @loginStarting.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер готовит вход…'**
  String get loginStarting;

  /// No description provided for @loginStartingHint.
  ///
  /// In ru, this message translates to:
  /// **'Обычно это занимает несколько секунд.'**
  String get loginStartingHint;

  /// No description provided for @loginUrlPending.
  ///
  /// In ru, this message translates to:
  /// **'Ждём ссылку на страницу входа.'**
  String get loginUrlPending;

  /// No description provided for @deviceCodeTitle.
  ///
  /// In ru, this message translates to:
  /// **'Код для входа'**
  String get deviceCodeTitle;

  /// No description provided for @deviceCodeSteps.
  ///
  /// In ru, this message translates to:
  /// **'1. Скопируйте код.\n2. Откройте страницу входа и войдите в ChatGPT.\n3. Введите код и подтвердите доступ.'**
  String get deviceCodeSteps;

  /// No description provided for @claudeSteps.
  ///
  /// In ru, this message translates to:
  /// **'1. Откройте страницу входа и войдите в Claude.\n2. Подтвердите доступ и скопируйте показанный код.\n3. Вставьте код ниже.'**
  String get claudeSteps;

  /// No description provided for @openLoginPage.
  ///
  /// In ru, this message translates to:
  /// **'Открыть страницу входа'**
  String get openLoginPage;

  /// No description provided for @openFailed.
  ///
  /// In ru, this message translates to:
  /// **'Не удалось открыть браузер'**
  String get openFailed;

  /// No description provided for @pasteCode.
  ///
  /// In ru, this message translates to:
  /// **'Вставьте код со страницы Claude'**
  String get pasteCode;

  /// No description provided for @sendCode.
  ///
  /// In ru, this message translates to:
  /// **'Отправить код'**
  String get sendCode;

  /// No description provided for @codeInvalid.
  ///
  /// In ru, this message translates to:
  /// **'Вставьте код целиком, без пробелов.'**
  String get codeInvalid;

  /// No description provided for @loginWaiting.
  ///
  /// In ru, this message translates to:
  /// **'Ждём подтверждения'**
  String get loginWaiting;

  /// No description provided for @loginWaitingHint.
  ///
  /// In ru, this message translates to:
  /// **'Экран обновится автоматически после входа.'**
  String get loginWaitingHint;

  /// No description provided for @loginOnComputer.
  ///
  /// In ru, this message translates to:
  /// **'Войдите на компьютере'**
  String get loginOnComputer;

  /// No description provided for @loginOnComputerHint.
  ///
  /// In ru, this message translates to:
  /// **'На компьютере открылся браузер. Войдите в аккаунт {service} — этот экран обновится сам.'**
  String loginOnComputerHint(String service);

  /// No description provided for @loginVerifying.
  ///
  /// In ru, this message translates to:
  /// **'Проверяем вход…'**
  String get loginVerifying;

  /// No description provided for @loginVerifyingHint.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер сохраняет аккаунт.'**
  String get loginVerifyingHint;

  /// No description provided for @loginCompleted.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунт подключён'**
  String get loginCompleted;

  /// No description provided for @loginCompletedHint.
  ///
  /// In ru, this message translates to:
  /// **'Он выбран активным для новых задач.'**
  String get loginCompletedHint;

  /// No description provided for @loginFailed.
  ///
  /// In ru, this message translates to:
  /// **'Вход не выполнен'**
  String get loginFailed;

  /// No description provided for @loginExpired.
  ///
  /// In ru, this message translates to:
  /// **'Время на вход истекло'**
  String get loginExpired;

  /// No description provided for @loginCancelled.
  ///
  /// In ru, this message translates to:
  /// **'Вход отменён'**
  String get loginCancelled;

  /// No description provided for @loginFailedHint.
  ///
  /// In ru, this message translates to:
  /// **'Попробуйте ещё раз или выберите вход на компьютере.'**
  String get loginFailedHint;

  /// No description provided for @loginExpires.
  ///
  /// In ru, this message translates to:
  /// **'Вход нужно завершить до {time}'**
  String loginExpires(String time);

  /// No description provided for @cancelLogin.
  ///
  /// In ru, this message translates to:
  /// **'Отменить вход'**
  String get cancelLogin;

  /// No description provided for @deviceCodeUnavailable.
  ///
  /// In ru, this message translates to:
  /// **'ChatGPT не разрешает вход по коду для этого аккаунта. Включите его в ChatGPT → Settings → Security или выберите «На компьютере».'**
  String get deviceCodeUnavailable;

  /// No description provided for @codeRejected.
  ///
  /// In ru, this message translates to:
  /// **'Код не принят. Начните вход заново и вставьте новый код.'**
  String get codeRejected;

  /// No description provided for @cliMissingShort.
  ///
  /// In ru, this message translates to:
  /// **'CLI не найден на компьютере.'**
  String get cliMissingShort;

  /// No description provided for @loginTimeout.
  ///
  /// In ru, this message translates to:
  /// **'Вход не завершён вовремя. Начните заново.'**
  String get loginTimeout;

  /// No description provided for @ownerRole.
  ///
  /// In ru, this message translates to:
  /// **'Владелец установки'**
  String get ownerRole;

  /// No description provided for @connection.
  ///
  /// In ru, this message translates to:
  /// **'Подключение'**
  String get connection;

  /// No description provided for @installation.
  ///
  /// In ru, this message translates to:
  /// **'Установка'**
  String get installation;

  /// No description provided for @deviceSessions.
  ///
  /// In ru, this message translates to:
  /// **'Устройства'**
  String get deviceSessions;

  /// No description provided for @noDevices.
  ///
  /// In ru, this message translates to:
  /// **'Нет данных об устройствах'**
  String get noDevices;

  /// No description provided for @unknownDevice.
  ///
  /// In ru, this message translates to:
  /// **'Устройство'**
  String get unknownDevice;

  /// No description provided for @thisDevice.
  ///
  /// In ru, this message translates to:
  /// **'Это устройство'**
  String get thisDevice;

  /// No description provided for @deviceSeen.
  ///
  /// In ru, this message translates to:
  /// **'Активность: {ago}'**
  String deviceSeen(String ago);

  /// No description provided for @deviceNeverSeen.
  ///
  /// In ru, this message translates to:
  /// **'Активности ещё не было'**
  String get deviceNeverSeen;

  /// No description provided for @deviceRevokedLabel.
  ///
  /// In ru, this message translates to:
  /// **'Доступ отозван'**
  String get deviceRevokedLabel;

  /// No description provided for @revoke.
  ///
  /// In ru, this message translates to:
  /// **'Отозвать доступ'**
  String get revoke;

  /// No description provided for @revokeTitle.
  ///
  /// In ru, this message translates to:
  /// **'Отозвать доступ?'**
  String get revokeTitle;

  /// No description provided for @revokeHint.
  ///
  /// In ru, this message translates to:
  /// **'Устройство «{device}» потеряет доступ и должно будет войти заново.'**
  String revokeHint(String device);

  /// No description provided for @theme.
  ///
  /// In ru, this message translates to:
  /// **'Оформление'**
  String get theme;

  /// No description provided for @systemTheme.
  ///
  /// In ru, this message translates to:
  /// **'Системная'**
  String get systemTheme;

  /// No description provided for @lightTheme.
  ///
  /// In ru, this message translates to:
  /// **'Светлая'**
  String get lightTheme;

  /// No description provided for @darkTheme.
  ///
  /// In ru, this message translates to:
  /// **'Тёмная'**
  String get darkTheme;

  /// No description provided for @about.
  ///
  /// In ru, this message translates to:
  /// **'О приложении'**
  String get about;

  /// No description provided for @appVersion.
  ///
  /// In ru, this message translates to:
  /// **'Версия'**
  String get appVersion;

  /// No description provided for @security.
  ///
  /// In ru, this message translates to:
  /// **'Безопасность'**
  String get security;

  /// No description provided for @securityValue.
  ///
  /// In ru, this message translates to:
  /// **'Сессия в защищённом хранилище Android · ключи ИИ остаются на компьютере'**
  String get securityValue;

  /// No description provided for @logout.
  ///
  /// In ru, this message translates to:
  /// **'Выйти'**
  String get logout;

  /// No description provided for @logoutTitle.
  ///
  /// In ru, this message translates to:
  /// **'Выйти из аккаунта?'**
  String get logoutTitle;

  /// No description provided for @logoutHint.
  ///
  /// In ru, this message translates to:
  /// **'Сессия на этом телефоне будет завершена. Чтобы снова открыть разделы, потребуется войти.'**
  String get logoutHint;

  /// No description provided for @logoutFootnote.
  ///
  /// In ru, this message translates to:
  /// **'Задачи и аккаунты ИИ на компьютере не удаляются.'**
  String get logoutFootnote;

  /// No description provided for @ownerAlreadyBound.
  ///
  /// In ru, this message translates to:
  /// **'Владелец уже зарегистрирован. Войдите с его email и паролем.'**
  String get ownerAlreadyBound;

  /// No description provided for @ownerRequired.
  ///
  /// In ru, this message translates to:
  /// **'Этот аккаунт не совпадает с владельцем установки.'**
  String get ownerRequired;

  /// No description provided for @registrationUncertain.
  ///
  /// In ru, this message translates to:
  /// **'Исход регистрации не подтверждён. Проверьте письмо перед повтором.'**
  String get registrationUncertain;

  /// No description provided for @registrationConflict.
  ///
  /// In ru, this message translates to:
  /// **'Регистрация уже начата. Подтвердите ранее указанный email и войдите.'**
  String get registrationConflict;

  /// No description provided for @rateLimited.
  ///
  /// In ru, this message translates to:
  /// **'Слишком много попыток. Подождите минуту и повторите.'**
  String get rateLimited;

  /// No description provided for @invalidUrl.
  ///
  /// In ru, this message translates to:
  /// **'Нужен HTTPS-адрес без пути, параметров и localhost.'**
  String get invalidUrl;

  /// No description provided for @networkError.
  ///
  /// In ru, this message translates to:
  /// **'Не удалось связаться с сервером. Проверьте интернет и доступность компьютера.'**
  String get networkError;

  /// No description provided for @responseError.
  ///
  /// In ru, this message translates to:
  /// **'Сервер вернул неподдерживаемый ответ.'**
  String get responseError;

  /// No description provided for @storageError.
  ///
  /// In ru, this message translates to:
  /// **'Не удалось открыть защищённое хранилище. Перезапустите приложение.'**
  String get storageError;

  /// No description provided for @notConfirmed.
  ///
  /// In ru, this message translates to:
  /// **'Сначала подтвердите email по ссылке из письма.'**
  String get notConfirmed;

  /// No description provided for @setupError.
  ///
  /// In ru, this message translates to:
  /// **'Неверный код настройки.'**
  String get setupError;

  /// No description provided for @errorOffline.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер не на связи. Запустите мост Portway на компьютере и повторите.'**
  String get errorOffline;

  /// No description provided for @disabled.
  ///
  /// In ru, this message translates to:
  /// **'Подключение ИИ на сервере ещё не настроено.'**
  String get disabled;

  /// No description provided for @busyError.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер уже выполняет другую задачу. Дождитесь её завершения.'**
  String get busyError;

  /// No description provided for @errorLimit.
  ///
  /// In ru, this message translates to:
  /// **'Лимит выбранного ИИ исчерпан.'**
  String get errorLimit;

  /// No description provided for @errorAccountRequired.
  ///
  /// In ru, this message translates to:
  /// **'Для выбранного ИИ нет аккаунта с выполненным входом.'**
  String get errorAccountRequired;

  /// No description provided for @errorProviderUnavailable.
  ///
  /// In ru, this message translates to:
  /// **'Выбранный ИИ не установлен на компьютере.'**
  String get errorProviderUnavailable;

  /// No description provided for @errorLoginActive.
  ///
  /// In ru, this message translates to:
  /// **'Уже идёт другой вход в аккаунт.'**
  String get errorLoginActive;

  /// No description provided for @errorAccountSystem.
  ///
  /// In ru, this message translates to:
  /// **'Системный аккаунт компьютера нельзя удалить из приложения.'**
  String get errorAccountSystem;

  /// No description provided for @errorInvalidRequest.
  ///
  /// In ru, this message translates to:
  /// **'Сервер не принял параметры запроса.'**
  String get errorInvalidRequest;

  /// No description provided for @eventGap.
  ///
  /// In ru, this message translates to:
  /// **'Часть событий пропущена. Потяните вниз, чтобы обновить.'**
  String get eventGap;

  /// No description provided for @notFound.
  ///
  /// In ru, this message translates to:
  /// **'Не найдено. Возможно, сервер ещё не обновлён.'**
  String get notFound;

  /// No description provided for @serverError.
  ///
  /// In ru, this message translates to:
  /// **'Не удалось выполнить запрос. Попробуйте ещё раз.'**
  String get serverError;

  /// No description provided for @errorLoginNotActive.
  ///
  /// In ru, this message translates to:
  /// **'Этот вход уже завершён или отменён. Начните вход заново.'**
  String get errorLoginNotActive;

  /// No description provided for @errorCommandsBusy.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер обрабатывает много команд. Повторите через минуту.'**
  String get errorCommandsBusy;

  /// No description provided for @commandExpired.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер не забрал команду вовремя. Проверьте, что мост Portway запущен.'**
  String get commandExpired;

  /// No description provided for @accountNotFound.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунт не найден на компьютере. Обновите список.'**
  String get accountNotFound;

  /// No description provided for @accountBusy.
  ///
  /// In ru, this message translates to:
  /// **'Аккаунт занят выполняемой задачей. Дождитесь её завершения.'**
  String get accountBusy;

  /// No description provided for @commandFailedGeneric.
  ///
  /// In ru, this message translates to:
  /// **'Компьютер не выполнил команду. Попробуйте ещё раз.'**
  String get commandFailedGeneric;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['ru'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'ru':
      return AppLocalizationsRu();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
