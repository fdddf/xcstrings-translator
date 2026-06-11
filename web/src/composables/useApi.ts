import { ref } from 'vue'

export interface App {
  id: number;
  name: string;
  description?: string;
  bundleId: string;
  appleId?: string;
  primaryLocale: string;
  shortDescription?: string;
  longDescription?: string;
  keywords?: string;
  supportUrl?: string;
  marketingUrl?: string;
  privacyUrl?: string;
  version?: string;
  appCategory?: string;
  isReadyForReview?: boolean;
  userId: number;
  createdAt: string;
  updatedAt: string;
  // Additional fields for UI
  platform?: string;
  origin?: string; // 'manual' or 'synced'
  synced?: boolean;
  sourceLanguage?: string;
  projectId?: number;
}

export interface AppLocalization {
  id: number;
  appId: number;
  languageCode: string;
  name?: string;
  subtitle?: string;
  privacyUrl?: string;
  marketingUrl?: string;
  supportUrl?: string;
  downloadDescription?: string;
  description?: string;
  shortDescription?: string;
  longDescription?: string;
  keywords?: string;
  whatsNew?: string;
  promotionalText?: string;
  syncedAt?: string;
  source?: string;
  syncStatus?: string;
  version?: string;
  versionState?: string;
  createdAt: string;
  updatedAt: string;
}

export interface User {
  id: number;
  username: string;
  email: string;
  isActive: boolean;
  isActivated: boolean;
  isSubscribed: boolean;
  subscriptionType: string;
  maxApps: number;
  maxTranslations: number;
  currentUsage: number;
  currentAppCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface Project {
  id: number;
  name: string;
  description?: string;
  projectType?: string;
  fileName?: string;
  fileContent?: string;
  sourceLanguage?: string;
  contentStructure?: any;
  settings?: any;
  userId: number;
  createdAt: string;
  updatedAt: string;
}

export interface Translation {
  id: number;
  projectId: number;
  key: string;
  sourceText: string;
  targetText?: string;
  targetLanguage: string;
  state?: string;
  translationProvider?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ProviderConfig {
  id: number;
  userId: number;
  providerType: string;
  configData: any;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface UserActivity {
  id: number;
  userId: number;
  action: string;
  details?: string;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Subscription {
  id: number;
  userId: number;
  stripeSubscriptionId?: string;
  stripeCustomerId?: string;
  subscriptionType: string;
  subscriptionStatus: string;
  currentPeriodStart: string;
  currentPeriodEnd: string;
  trialEnd?: string;
  cancelAtPeriodEnd: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface TranslationQueue {
  id: number;
  userId: number;
  projectId?: number;
  appId?: number;
  type: string;
  status: string;
  priority: number;
  progress: number;
  total: number;
  done: number;
  error?: string;
  providerType: string;
  sourceLanguage: string;
  targetLanguages: string[];
  configData: any;
  resultData?: any;
  createdAt: string;
  updatedAt: string;
}

export interface ApiResponse<T> {
  success: boolean;
  message?: string;
  data?: T;
  error?: string;
}

// Define types for API responses
export interface AppsResponse {
  success: boolean;
  apps: App[];
}

export interface AppResponse {
  success: boolean;
  app: App;
}

export interface AppLocalizationsResponse {
  success: boolean;
  localizations: AppLocalization[];
}

export interface AppLocalizationResponse {
  success: boolean;
  localization: AppLocalization;
}

export interface UserResponse {
  success: boolean;
  user: User;
}

export interface ProjectsResponse {
  success: boolean;
  projects: Project[];
}

export interface ProjectResponse {
  success: boolean;
  project: Project;
}

export interface TranslationsResponse {
  success: boolean;
  translations: Translation[];
}

export interface ProviderConfigsResponse {
  success: boolean;
  configs: ProviderConfig[];
}

export interface ProviderConfigResponse {
  success: boolean;
  config: ProviderConfig;
}

export interface UserActivitiesResponse {
  success: boolean;
  activities: UserActivity[];
}

export interface SubscriptionResponse {
  success: boolean;
  user: User;
}

export interface UsageResponse {
  success: boolean;
  overLimit: boolean;
  usage: number;
  limit: number;
  percentage: number;
}

export interface QueueJobResponse {
  success: boolean;
  job: TranslationQueue;
}

export interface QueueJobsResponse {
  success: boolean;
  jobs: TranslationQueue[];
}

export interface QueueJobCreateRequest {
  jobType: string;
  projectId?: number;
  appId?: number;
  providerType: string;
  sourceLanguage: string;
  targetLanguages: string[];
  onlyTranslateWhatsNew?: boolean;
  configData?: Record<string, any>;
}

export interface LanguageMetadata {
  code: string;
  name: string;
  native_name: string;
  region?: string;
  direction: string;
  emoji?: string;
}

export interface LanguagesResponse {
  success: boolean;
  languages: LanguageMetadata[];
}

export interface LoginResponse {
  success: boolean;
  message: string;
  user: User;
  token: string;
}

export interface RegisterResponse {
  success: boolean;
  message: string;
  user?: User;
}

class ApiClient {
  private baseUrl = '/api';
  private token = ref<string | null>(null);

  constructor() {
    // Try to get token from localStorage
    const storedToken = localStorage.getItem('token');
    if (storedToken) {
      this.token.value = storedToken;
    }
  }

  // Set authentication token
  setToken(token: string) {
    this.token.value = token;
    localStorage.setItem('token', token);
  }

  // Clear authentication token
  clearToken() {
    this.token.value = null;
    localStorage.removeItem('token');
  }

  // Make an HTTP request
  async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;

    const isFormData = options.body instanceof FormData
    const headers = {
      ...(isFormData ? {} : { 'Content-Type': 'application/json' }),
      ...options.headers,
    } as Record<string, string>;

    if (this.token.value) {
      headers['Authorization'] = `Bearer ${this.token.value}`;
    }

    const config: RequestInit = {
      ...options,
      headers,
    };

    const response = await fetch(url, config);

    if (!response.ok) {
      const errorData = await response.text();
      let errorMessage: string;

      // Try to parse as JSON to extract the message
      try {
        const parsedError = JSON.parse(errorData);
        errorMessage = parsedError.message || errorData || `HTTP error! status: ${response.status}`;
      } catch {
        // If parsing fails, use the raw error text
        errorMessage = errorData || `HTTP error! status: ${response.status}`;
      }

      // Handle token expiration (401 Unauthorized)
      if (response.status === 401 || errorMessage.toLowerCase().includes('invalid or expired token')) {
        this.clearToken();
        localStorage.removeItem('currentUser');
        // Dispatch custom event for token expiration
        window.dispatchEvent(new CustomEvent('token-expired'));
      }

      throw new Error(errorMessage);
    }

    const data = await response.json();
    return data;
  }

  // Auth methods
  async login(username: string, password: string): Promise<LoginResponse> {
    const response = await this.request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });

    // Set token on successful login
    if (response.success && response.token) {
      this.setToken(response.token);
    }

    return response;
  }

  async register(username: string, email: string, password: string): Promise<RegisterResponse> {
    return this.request<RegisterResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    });
  }

  async logout(): Promise<any> {
    return this.request('/auth/logout', {
      method: 'POST',
    });
  }

  // App methods
  async getApps(): Promise<AppsResponse> {
    return this.request<AppsResponse>('/protected/apps');
  }

  async getAppsByProject(projectId: number): Promise<AppsResponse> {
    return this.request<AppsResponse>(`/protected/apps?projectId=${projectId}`);
  }

  async createApp(appData: Partial<App>): Promise<AppResponse> {
    return this.request<AppResponse>('/protected/apps', {
      method: 'POST',
      body: JSON.stringify(appData),
    });
  }

  async getApp(appId: number): Promise<AppResponse> {
    return this.request<AppResponse>(`/protected/apps/${appId}`);
  }

  async updateApp(appId: number, appData: Partial<App>): Promise<any> {
    return this.request(`/protected/apps/${appId}`, {
      method: 'PUT',
      body: JSON.stringify(appData),
    });
  }

  async deleteApp(appId: number): Promise<any> {
    return this.request(`/protected/apps/${appId}`, {
      method: 'DELETE',
    });
  }

  // App localization methods
  async getAppLocalizations(appId: number): Promise<AppLocalizationsResponse> {
    return this.request<AppLocalizationsResponse>(`/protected/apps/${appId}/localizations`);
  }

  async createAppLocalization(appId: number, localizationData: Partial<AppLocalization>): Promise<AppLocalizationResponse> {
    return this.request<AppLocalizationResponse>(`/protected/apps/${appId}/localizations`, {
      method: 'POST',
      body: JSON.stringify(localizationData),
    });
  }

  async getAppLocalization(appId: number, languageCode: string): Promise<AppLocalizationResponse> {
    return this.request<AppLocalizationResponse>(`/protected/apps/${appId}/localizations/${languageCode}`);
  }

  async updateAppLocalization(appId: number, languageCode: string, localizationData: Partial<AppLocalization>): Promise<any> {
    return this.request(`/protected/apps/${appId}/localizations/${languageCode}`, {
      method: 'PUT',
      body: JSON.stringify(localizationData),
    });
  }

  async deleteAppLocalization(appId: number, languageCode: string): Promise<any> {
    return this.request(`/protected/apps/${appId}/localizations/${languageCode}`, {
      method: 'DELETE',
    });
  }

  // App user management methods
  async getAppUsers(appId: number): Promise<any> {
    return this.request(`/protected/apps/${appId}/users`);
  }

  async addUserToApp(appId: number, userData: { userId: number; role: string }): Promise<any> {
    return this.request(`/protected/apps/${appId}/users`, {
      method: 'POST',
      body: JSON.stringify(userData),
    });
  }

  async updateUserAppRole(appId: number, userId: number, role: string): Promise<any> {
    return this.request(`/protected/apps/${appId}/users/${userId}`, {
      method: 'PUT',
      body: JSON.stringify({ role }),
    });
  }

  async removeUserFromApp(appId: number, userId: number): Promise<any> {
    return this.request(`/protected/apps/${appId}/users/${userId}`, {
      method: 'DELETE',
    });
  }

  async getAppLanguages(appId: number): Promise<any> {
    return this.request(`/protected/apps/${appId}/languages`);
  }

  async addAppLanguage(appId: number, language: string): Promise<any> {
    return this.request(`/protected/apps/${appId}/languages`, {
      method: 'POST',
      body: JSON.stringify({ language }),
    });
  }

  async removeAppLanguage(appId: number, language: string): Promise<any> {
    return this.request(`/protected/apps/${appId}/languages/${language}`, {
      method: 'DELETE',
    });
  }

  // Project methods
  async getProjects(projectType?: string): Promise<ProjectsResponse> {
    const query = projectType ? `?type=${encodeURIComponent(projectType)}` : '';
    return this.request<ProjectsResponse>(`/protected/projects${query}`);
  }

  async createProject(projectData: Partial<Project>): Promise<ProjectResponse> {
    return this.request<ProjectResponse>('/protected/projects', {
      method: 'POST',
      body: JSON.stringify(projectData),
    });
  }

  async getProject(projectId: number): Promise<ProjectResponse> {
    return this.request<ProjectResponse>(`/protected/projects/${projectId}`);
  }

  async updateProject(projectId: number, projectData: Partial<Project>): Promise<any> {
    return this.request(`/protected/projects/${projectId}`, {
      method: 'PUT',
      body: JSON.stringify(projectData),
    });
  }

  async deleteProject(projectId: number): Promise<any> {
    return this.request(`/protected/projects/${projectId}`, {
      method: 'DELETE',
    });
  }

  // Project upload and translation methods
  async uploadToProject(projectId: number, file: File, sourceLanguage?: string): Promise<any> {
    const formData = new FormData();
    formData.append('file', file);
    if (sourceLanguage) {
      formData.append('sourceLanguage', sourceLanguage);
    }
    return this.request(`/protected/projects/${projectId}/upload`, {
      method: 'POST',
      body: formData,
    });
  }

  async createProjectFromFile(file: File, sourceLanguage?: string): Promise<ProjectResponse> {
    const fileContent = await file.text();
    return this.createProject({
      name: file.name,
      description: '',
      fileName: file.name,
      fileContent,
      sourceLanguage,
      projectType: 'xcstrings'
    });
  }

  async translateProject(projectId: number, translateData: any): Promise<any> {
    return this.request(`/protected/projects/${projectId}/translate`, {
      method: 'POST',
      body: JSON.stringify(translateData),
    });
  }

  async exportProject(projectId: number): Promise<Blob> {
    const response = await fetch(`${this.baseUrl}/protected/projects/${projectId}/export`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${this.token.value}`,
      },
    });
    return response.blob();
  }

  // Translation methods
  async getTranslations(projectId: number): Promise<TranslationsResponse> {
    return this.request<TranslationsResponse>(`/protected/projects/${projectId}/translations`);
  }

  async getMissingTranslations(projectId: number, targetLanguages: string[]): Promise<any> {
    return this.request(`/protected/projects/${projectId}/missing-translations`, {
      method: 'POST',
      body: JSON.stringify({ targetLanguages }),
    });
  }

  async getTranslationStatus(projectId: number, targetLanguages: string[]): Promise<any> {
    const params = new URLSearchParams();
    targetLanguages.forEach(lang => params.append('targetLanguages', lang));
    return this.request(`/protected/projects/${projectId}/translation-status?${params.toString()}`);
  }

  // Provider config methods
  async getProviderConfigs(): Promise<ProviderConfigsResponse> {
    return this.request<ProviderConfigsResponse>('/protected/providers');
  }

  async createProviderConfig(configData: Partial<ProviderConfig>): Promise<ProviderConfigResponse> {
    return this.request<ProviderConfigResponse>('/protected/providers', {
      method: 'POST',
      body: JSON.stringify(configData),
    });
  }

  async getProviderConfig(configId: number): Promise<ProviderConfigResponse> {
    return this.request<ProviderConfigResponse>(`/protected/providers/${configId}`);
  }

  async updateProviderConfig(configId: number, configData: Partial<ProviderConfig>): Promise<any> {
    return this.request(`/protected/providers/${configId}`, {
      method: 'PUT',
      body: JSON.stringify(configData),
    });
  }

  async deleteProviderConfig(configId: number): Promise<any> {
    return this.request(`/protected/providers/${configId}`, {
      method: 'DELETE',
    });
  }

  async getDefaultProviderConfig(providerType: string): Promise<ProviderConfigResponse> {
    return this.request<ProviderConfigResponse>(`/protected/providers/${providerType}/default`);
  }

  // Language methods
  async getSupportedLanguages(): Promise<LanguagesResponse> {
    return this.request<LanguagesResponse>('/protected/languages');
  }

  async translateText(text: string, sourceLanguage: string, targetLanguage: string, providerConfigId?: number): Promise<any> {
    return this.request('/protected/translate/text', {
      method: 'POST',
      body: JSON.stringify({
        text,
        sourceLanguage,
        targetLanguage,
        providerConfigId
      }),
    });
  }

  // Activity methods
  async getUserActivities(): Promise<UserActivitiesResponse> {
    return this.request<UserActivitiesResponse>('/protected/activities');
  }

  async getAdminActivities(): Promise<UserActivitiesResponse> {
    return this.request<UserActivitiesResponse>('/protected/activities/admin');
  }

  // User management methods
  async getUsers(): Promise<any> {
    return this.request('/protected/users');
  }

  async getUser(userId: number): Promise<UserResponse> {
    return this.request<UserResponse>(`/protected/users/${userId}`);
  }

  async updateUser(userId: number, userData: Partial<User>): Promise<any> {
    return this.request(`/protected/users/${userId}`, {
      method: 'PUT',
      body: JSON.stringify(userData),
    });
  }

  async deleteUser(userId: number): Promise<any> {
    return this.request(`/protected/users/${userId}`, {
      method: 'DELETE',
    });
  }

  async activateUser(userId: number): Promise<any> {
    return this.request(`/protected/users/${userId}/activate`, {
      method: 'POST',
    });
  }

  async deactivateUser(userId: number): Promise<any> {
    return this.request(`/protected/users/${userId}/deactivate`, {
      method: 'POST',
    });
  }

  // Subscription methods
  async getUserSubscription(): Promise<SubscriptionResponse> {
    return this.request<SubscriptionResponse>('/protected/subscription');
  }

  async getUsage(): Promise<UsageResponse> {
    return this.request<UsageResponse>('/protected/subscription/usage');
  }

  async handleSubscriptionWebhook(webhookData: any): Promise<any> {
    return this.request('/protected/subscription/webhook', {
      method: 'POST',
      body: JSON.stringify(webhookData),
    });
  }

  // Queue methods
  async createQueueJob(jobData: QueueJobCreateRequest): Promise<QueueJobResponse> {
    return this.request<QueueJobResponse>('/protected/queue/translate', {
      method: 'POST',
      body: JSON.stringify(jobData),
    });
  }

  async getQueueJobs(): Promise<QueueJobsResponse> {
    return this.request<QueueJobsResponse>('/protected/queue/jobs');
  }

  async getQueueJob(jobId: number): Promise<QueueJobResponse> {
    return this.request<QueueJobResponse>(`/protected/queue/jobs/${jobId}`);
  }

  // App localization translation methods
  async translateAppLocalizations(appId: number, translateData: {
    providerType: string;
    providerConfigId?: number;
    sourceLanguage: string;
    targetLanguages: string[];
    onlyTranslateWhatsNew?: boolean;
    configData: any;
  }): Promise<QueueJobResponse> {
    return this.request<QueueJobResponse>('/protected/queue/translate', {
      method: 'POST',
      body: JSON.stringify({
        jobType: 'app_localization',
        appId: appId,
        ...translateData,
      }),
    });
  }

  // Project additional methods
  async getProjectStats(projectId: number): Promise<any> {
    return this.request(`/protected/projects/${projectId}/stats`);
  }

  async updateSingleTranslation(projectId: number, key: string, language: string, data: { targetText: string; state?: string }): Promise<any> {
    return this.request(`/protected/projects/${projectId}/translations/${key}/${language}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async bulkUpdateTranslations(projectId: number, updates: any[]): Promise<any> {
    return this.request(`/protected/projects/${projectId}/translations/bulk`, {
      method: 'PUT',
      body: JSON.stringify({ updates }),
    });
  }

  async getProjectLanguages(projectId: number): Promise<any> {
    return this.request(`/protected/projects/${projectId}/languages`);
  }

  // App additional methods
  async getAppStats(appId: number): Promise<any> {
    return this.request(`/protected/apps/${appId}/stats`);
  }

  async bulkUpdateAppLocalizations(appId: number, updates: any[]): Promise<any> {
    return this.request(`/protected/apps/${appId}/localizations/bulk`, {
      method: 'PUT',
      body: JSON.stringify({ updates }),
    });
  }

  // Apple Connect methods
  async syncAppleApps(payload: { configId: number }): Promise<any> {
    return this.request('/protected/apple-connect/sync-apps', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  // Apple Connect configuration methods
  async getAppleConnectConfigs(): Promise<any> {
    return this.request('/protected/appleconnections');
  }

  async getAppleConnectConfig(configId: number): Promise<any> {
    return this.request(`/protected/appleconnections/${configId}`);
  }

  async createAppleConnectConfig(configData: { issuerId: string; keyId: string; privateKey: string; isDefault?: boolean }): Promise<any> {
    return this.request('/protected/appleconnections', {
      method: 'POST',
      body: JSON.stringify(configData),
    });
  }

  async updateAppleConnectConfig(configId: number, configData: { issuerId?: string; keyId?: string; privateKey?: string; isDefault?: boolean }): Promise<any> {
    return this.request(`/protected/appleconnections/${configId}`, {
      method: 'PUT',
      body: JSON.stringify(configData),
    });
  }

  async deleteAppleConnectConfig(configId: number): Promise<any> {
    return this.request(`/protected/appleconnections/${configId}`, {
      method: 'DELETE',
    });
  }

  async testAppleConnectConnection(configId: number): Promise<any> {
    return this.request(`/protected/appleconnections/${configId}/test`, {
      method: 'POST',
    });
  }

  async testAppleConnectCredentials(credentials: { issuerId: string; keyId: string; privateKey: string }): Promise<any> {
    return this.request('/protected/appleconnections/test', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
  }

  async syncAppleAppLocalizations(appId: number, payload: { configId: number; languageCodes?: string[]; direction?: string; strategy?: string }): Promise<any> {
    return this.request(`/protected/apple-connect/${appId}/sync-localizations`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async syncAppToApple(appId: number, payload: { configId: number; languageCode?: string }): Promise<any> {
    return this.request(`/protected/apps/${appId}/sync-to-apple`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async pushUpdateContentToApple(appId: number, payload: { configId: number; languageCodes?: string[] }): Promise<any> {
    return this.request(`/protected/apple-connect/${appId}/push-update-content`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  }

const apiClient = new ApiClient();

export function useApi() {
  return {
    api: apiClient,
  };
}
