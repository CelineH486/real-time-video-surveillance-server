class ApiConfig {
  static const defaultBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://localhost:8080',
  );

  static const defaultApiToken = String.fromEnvironment(
    'API_TOKEN',
    defaultValue: 'dev-user-token',
  );
}
