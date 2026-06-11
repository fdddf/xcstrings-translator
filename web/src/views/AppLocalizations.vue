<template>
  <div class="space-y-6">
    <header class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <div>
        <p class="text-xs uppercase tracking-[0.3em] text-slate-500">{{ t('nav.applocalizations') }}</p>
        <h1 class="text-2xl font-semibold">{{ t('applocalizations.title') }}</h1>
      </div>
      <div class="flex items-center gap-2">
        <button
          class="rounded-lg border border-white/20 px-3 py-2 text-sm hover:border-mint/60 hover:text-mint"
          @click="showBatchTranslateModal = false; showUpdateContentModal = true"
          :disabled="batchTranslating"
        >
          {{ t('applocalizations.syncUpdateContent') }}
        </button>
        <button
          class="rounded-lg border border-white/20 px-3 py-2 text-sm hover:border-mint/60 hover:text-mint"
          @click="showBatchTranslateModal = true"
          :disabled="batchTranslating"
        >
          {{ batchTranslating ? t('applocalizations.translateAllProgress', { progress: batchTranslateProgress }) : t('applocalizations.batchTranslate') }}
        </button>
        <button
          class="rounded-lg border border-white/20 px-3 py-2 text-sm hover:border-mint/60 hover:text-mint"
          @click="showSyncModal = true"
        >
          {{ t('applocalizations.sync') }}
        </button>
      </div>
    </header>

    <!-- Main Content with Two Columns -->
    <div class="flex gap-6">
      <!-- Left: Language Sidebar -->
      <aside class="w-80 flex-shrink-0">
        <div class="sticky top-6 rounded-2xl border border-white/10 bg-white/5 p-4">
          <h2 class="mb-4 text-sm font-semibold text-slate-300">{{ t('applocalizations.languages') }}</h2>
          
          <!-- Language Dropdown for Adding -->
          <div class="mb-4">
            <select
              v-model="selectedLanguageToAdd"
              class="w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint"
              @change="handleLanguageSelect"
            >
              <option value="">{{ t('applocalizations.selectLanguage') }}</option>
              <option v-for="lang in availableLanguages" :key="lang.code" :value="lang.code">
                {{ getLanguageFlag(lang.code) }} {{ lang.name }} ({{ lang.code }})
              </option>
            </select>
          </div>
          
          <!-- Language List -->
          <div class="space-y-1 max-h-[calc(100vh-300px)] overflow-y-auto">
            <button
              v-for="loc in localizations"
              :key="loc.id"
              @click="selectedLocalizationId = loc.id"
              :class="[
                'w-full flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm transition-all',
                selectedLocalizationId === loc.id
                  ? 'bg-mint text-midnight font-medium'
                  : 'hover:bg-white/10 text-slate-300'
              ]"
            >
              <span class="text-lg">{{ getLanguageFlag(loc.languageCode) }}</span>
              <span class="flex-1 truncate">{{ getLanguageName(loc.languageCode) }}</span>
              <span v-if="loc.id < 0" class="text-xs text-orange-400">*</span>
              <span v-if="loc.syncStatus" :class="[
                'w-2 h-2 rounded-full flex-shrink-0',
                loc.syncStatus === 'synced' ? 'bg-green-500' :
                loc.syncStatus === 'pending' ? 'bg-yellow-500' :
                loc.syncStatus === 'failed' ? 'bg-red-500' : 'bg-gray-500'
              ]"></span>
            </button>
            
            <!-- Empty State -->
            <div v-if="localizations.length === 0" class="py-8 text-center text-sm text-slate-500">
              {{ t('applocalizations.noLanguages') }}
            </div>
          </div>
        </div>
      </aside>

    <!-- Sync Localizations Modal -->
    <div v-if="showSyncModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4">
      <div class="w-full max-w-lg rounded-2xl border border-white/10 bg-midnight p-6 shadow-xl">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">{{ t('applocalizations.sync') }}</h2>
          <button class="text-slate-400 hover:text-white" @click="closeSyncModal">×</button>
        </div>

        <div class="mt-4 space-y-4">
          <div>
            <p class="text-sm text-slate-400 mb-3">{{ t('applocalizations.selectProviderConfig') }}</p>
            <select
              v-model="selectedConfigId"
              class="w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint"
              required
            >
              <option value="">{{ t('applocalizations.chooseConfig') }}</option>
              <option v-for="config in appleConnectConfigs" :key="config.id" :value="config.id">
                {{ config.providerType }} ({{ getProviderDisplayName(config.providerType) }})
              </option>
            </select>
          </div>

          <div>
            <p class="text-sm text-slate-400 mb-3">{{ t('applocalizations.syncDirection') }}</p>
            <select
              v-model="syncDirection"
              class="w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint"
            >
              <option value="pull">{{ t('applocalizations.pullFromApple') }}</option>
              <option value="push">{{ t('applocalizations.pushToApple') }}</option>
              <option value="both">{{ t('applocalizations.bidirectional') }}</option>
            </select>
          </div>

          <div>
            <p class="text-sm text-slate-400 mb-3">{{ t('applocalizations.conflictStrategy') }}</p>
            <select
              v-model="syncStrategy"
              class="w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint"
            >
              <option value="apple_first">{{ t('applocalizations.appleFirst') }}</option>
              <option value="local_first">{{ t('applocalizations.localFirst') }}</option>
              <option value="manual">{{ t('applocalizations.manual') }}</option>
            </select>
          </div>

          <div>
            <div class="flex items-center justify-between mb-3">
              <p class="text-sm text-slate-400">{{ t('applocalizations.selectLanguages') }} ({{ t('applocalizations.optional') }})</p>
              <label class="flex items-center gap-2 text-sm cursor-pointer hover:text-mint text-slate-400">
                <input
                  type="checkbox"
                  :checked="selectAllSyncLanguages"
                  @change="toggleSelectAllSyncLanguages"
                  class="rounded bg-white/10 border-white/20 text-mint focus:ring-mint"
                />
                <span>{{ t('applocalizations.selectAll') || '全选' }}</span>
              </label>
            </div>
            <div class="max-h-40 overflow-y-auto rounded-lg bg-white/5 border border-white/10 p-2 space-y-1">
              <label v-for="lang in availableLanguages" :key="lang.code" class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded">
                <input
                  type="checkbox"
                  :value="lang.code"
                  v-model="selectedSyncLanguages"
                  class="rounded bg-white/10 border-white/20 text-mint focus:ring-mint"
                />
                <span>{{ lang.name }} ({{ lang.code }})</span>
              </label>
            </div>
            <p class="text-xs text-slate-500 mt-1">{{ t('applocalizations.selectLanguagesDesc') }}</p>
          </div>

          <div v-if="syncResult" class="p-3 rounded-lg bg-white/5 border border-white/10">
            <p class="text-sm">{{ syncResult.message }}</p>
            <p v-if="syncResult.count" class="text-sm text-mint">{{ t('applocalizations.syncedCount', { count: syncResult.count }) }}</p>
            <div v-if="syncResult.conflicts && syncResult.conflicts.length > 0" class="mt-2 p-2 rounded bg-yellow-500/10 border border-yellow-500/20">
              <p class="text-xs text-yellow-500">{{ t('applocalizations.conflictsDetected', { count: syncResult.conflicts.length }) }}</p>
            </div>
          </div>

          <div class="flex justify-end gap-2 pt-4">
            <button
              type="button"
              class="rounded-lg border border-white/20 px-3 py-2 text-sm hover:border-mint/60 hover:text-mint"
              @click="closeSyncModal"
            >
              {{ t('common.cancel') || 'Cancel' }}
            </button>
            <button
              type="button"
              class="rounded-lg bg-mint px-3 py-2 text-sm font-semibold text-midnight shadow"
              @click="syncLocalizations"
              :disabled="!selectedConfigId || syncing"
            >
              <span v-if="!syncing">{{ t('applocalizations.syncNow') }}</span>
              <span v-else class="flex items-center gap-1">
                <span class="h-3 w-3 rounded-full bg-midnight animate-pulse"></span>
                {{ t('applocalizations.syncing') }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Batch Translate Modal -->
    <div v-if="showBatchTranslateModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4">
      <div class="w-full max-w-lg rounded-2xl border border-white/10 bg-midnight p-6 shadow-xl">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">{{ t('applocalizations.batchTranslate') }}</h2>
          <button class="text-slate-400 hover:text-white" @click="closeBatchTranslateModal">×</button>
        </div>

        <div class="mt-4 space-y-4">
          <!-- Translation provider selection -->
          <div>
            <label class="text-sm text-slate-400">{{ t('applocalizations.translateProvider') }}</label>
            <select
              v-model="selectedTranslateProviderId"
              class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint"
            >
              <option :value="null">{{ t('applocalizations.localHunyuan') }}</option>
              <option v-for="config in translationProviderConfigs" :key="config.id" :value="config.id">
                {{ getProviderDisplayName(config.providerType) }}{{ config.isDefault ? ' ★' : '' }}
              </option>
            </select>
          </div>

          <div v-if="selectedTranslateProviderId === null" class="flex items-center gap-2 p-3 rounded-lg bg-mint/10 border border-mint/20">
            <span class="text-lg">🦙</span>
            <div>
              <p class="text-sm font-medium text-mint">腾讯混元 Hunyuan 模型</p>
              <p class="text-xs text-slate-400">本地模型翻译，数据安全</p>
            </div>
          </div>

          <!-- Overwrite option -->
          <div class="flex items-center gap-2 p-3 rounded-lg bg-white/5 border border-white/10">
            <input
              type="checkbox"
              id="overwriteExisting"
              v-model="overwriteExisting"
              class="rounded bg-white/10 border-white/20 text-mint focus:ring-mint"
            />
            <label for="overwriteExisting" class="text-sm cursor-pointer">
              <span class="text-slate-300">{{ t('applocalizations.overwriteExisting') }}</span>
              <p class="text-xs text-slate-500 mt-0.5">{{ t('applocalizations.overwriteExistingDesc') }}</p>
            </label>
          </div>

          <div>
            <div class="flex items-center justify-between mb-3">
              <p class="text-sm text-slate-400">{{ t('applocalizations.selectTargetLanguages') }}</p>
              <label class="flex items-center gap-2 text-sm cursor-pointer hover:text-mint text-slate-400">
                <input
                  type="checkbox"
                  :checked="selectAllLanguages"
                  @change="toggleSelectAllLanguages"
                  class="rounded bg-white/10 border-white/20 text-mint focus:ring-mint"
                />
                <span>{{ t('applocalizations.selectAll') || '全选' }}</span>
              </label>
            </div>
            <div class="max-h-60 overflow-y-auto rounded-lg bg-white/5 border border-white/10 p-2 space-y-1">
              <label v-for="lang in availableLanguages" :key="lang.code" class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded">
                <input
                  type="checkbox"
                  :value="lang.code"
                  v-model="selectedBatchTranslateLanguages"
                  :disabled="app?.primaryLocale === lang.code"
                  class="rounded bg-white/10 border-white/20 text-mint focus:ring-mint"
                />
                <span>{{ lang.name }} ({{ lang.code }})</span>
              </label>
            </div>
            <p class="text-xs text-slate-500 mt-1">{{ t('applocalizations.selectTargetLanguagesDesc') }}</p>
          </div>

          <div v-if="batchTranslateResult" class="p-3 rounded-lg bg-white/5 border border-white/10">
            <p class="text-sm">{{ batchTranslateResult.message }}</p>
            <div v-if="batchTranslateResult.progress !== undefined" class="mt-2">
              <div class="h-2 bg-white/10 rounded-full overflow-hidden">
                <div class="h-full bg-mint transition-all duration-300" :style="{ width: batchTranslateResult.progress + '%' }"></div>
              </div>
              <div class="flex justify-between text-xs text-slate-400 mt-1">
                <span>{{ t('applocalizations.translateAllProgress', { progress: batchTranslateResult.progress }) }}</span>
                <span v-if="batchTranslateResult.done !== undefined && batchTranslateResult.total !== undefined">
                  {{ batchTranslateResult.done }} / {{ batchTranslateResult.total }}
                </span>
              </div>
            </div>
          </div>

          <div class="flex justify-end gap-2 pt-4">
            <button
              type="button"
              class="rounded-lg border border-white/20 px-3 py-2 text-sm hover:border-mint/60 hover:text-mint"
              @click="closeBatchTranslateModal"
              :disabled="batchTranslating"
            >
              {{ t('common.cancel') || 'Cancel' }}
            </button>
            <button
              type="button"
              class="rounded-lg bg-mint px-3 py-2 text-sm font-semibold text-midnight shadow"
              @click="submitBatchTranslate"
              :disabled="selectedBatchTranslateLanguages.length === 0 || batchTranslating"
            >
              <span v-if="!batchTranslating">{{ t('applocalizations.translateAll') }}</span>
              <span v-else class="flex items-center gap-1">
                <span class="h-3 w-3 rounded-full bg-midnight animate-pulse"></span>
                {{ t('applocalizations.translateAllProgress', { progress: batchTranslateProgress }) }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Sync Update Content Modal (What's New + Promotional Text only) -->
    <div v-if="showUpdateContentModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4">
      <div class="w-full max-w-lg rounded-2xl border border-white/10 bg-midnight p-6 shadow-xl">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">{{ t('applocalizations.syncUpdateContent') }}</h2>
          <button class="text-slate-400 hover:text-white" @click="closeUpdateContentModal">×</button>
        </div>

        <div class="mt-4 space-y-4">
          <div class="flex items-center gap-2 p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
            <span class="text-lg">📝</span>
            <div>
              <p class="text-sm font-medium text-blue-400">{{ t('applocalizations.updateContentDesc') }}</p>
              <p class="text-xs text-slate-400">{{ t('applocalizations.updateContentFields') }}</p>
            </div>
          </div>

          <!-- Translation provider selection -->
          <div>
            <label class="text-sm text-slate-400">{{ t('applocalizations.translateProvider') }}</label>
            <select
              v-model="selectedTranslateProviderId"
              class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint"
            >
              <option :value="null">{{ t('applocalizations.localHunyuan') }}</option>
              <option v-for="config in translationProviderConfigs" :key="config.id" :value="config.id">
                {{ getProviderDisplayName(config.providerType) }}{{ config.isDefault ? ' ★' : '' }}
              </option>
            </select>
          </div>

          <div>
            <div class="flex items-center justify-between mb-3">
              <p class="text-sm text-slate-400">{{ t('applocalizations.selectTargetLanguages') }}</p>
              <label class="flex items-center gap-2 text-sm cursor-pointer hover:text-mint text-slate-400">
                <input
                  type="checkbox"
                  :checked="selectAllUpdateLanguages"
                  @change="toggleSelectAllUpdateLanguages"
                  class="rounded bg-white/10 border-white/20 text-mint focus:ring-mint"
                />
                <span>{{ t('applocalizations.selectAll') || '全选' }}</span>
              </label>
            </div>
            <div class="max-h-60 overflow-y-auto rounded-lg bg-white/5 border border-white/10 p-2 space-y-1">
              <label v-for="lang in availableLanguages" :key="lang.code" class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded">
                <input
                  type="checkbox"
                  :value="lang.code"
                  v-model="selectedUpdateContentLanguages"
                  :disabled="app?.primaryLocale === lang.code"
                  class="rounded bg-white/10 border-white/20 text-mint focus:ring-mint"
                />
                <span>{{ lang.name }} ({{ lang.code }})</span>
              </label>
            </div>
            <p class="text-xs text-slate-500 mt-1">{{ t('applocalizations.selectTargetLanguagesDesc') }}</p>
          </div>

          <div v-if="updateContentResult" class="p-3 rounded-lg bg-white/5 border border-white/10">
            <p class="text-sm">{{ updateContentResult.message }}</p>
            <div v-if="updateContentResult.progress !== undefined" class="mt-2">
              <div class="h-2 bg-white/10 rounded-full overflow-hidden">
                <div class="h-full bg-mint transition-all duration-300" :style="{ width: updateContentResult.progress + '%' }"></div>
              </div>
              <div class="flex justify-between text-xs text-slate-400 mt-1">
                <span>{{ t('applocalizations.translateAllProgress', { progress: updateContentResult.progress }) }}</span>
                <span v-if="updateContentResult.done !== undefined && updateContentResult.total !== undefined">
                  {{ updateContentResult.done }} / {{ updateContentResult.total }}
                </span>
              </div>
            </div>
          </div>

          <div class="flex justify-end gap-2 pt-4">
            <button
              type="button"
              class="rounded-lg border border-white/20 px-3 py-2 text-sm hover:border-mint/60 hover:text-mint"
              @click="closeUpdateContentModal"
              :disabled="updateContentTranslating"
            >
              {{ t('common.cancel') || 'Cancel' }}
            </button>
            <button
              type="button"
              class="rounded-lg bg-mint px-3 py-2 text-sm font-semibold text-midnight shadow"
              @click="submitUpdateContentTranslate"
              :disabled="selectedUpdateContentLanguages.length === 0 || updateContentTranslating"
            >
              <span v-if="!updateContentTranslating">{{ t('applocalizations.translateAll') }}</span>
              <span v-else class="flex items-center gap-1">
                <span class="h-3 w-3 rounded-full bg-midnight animate-pulse"></span>
                {{ t('applocalizations.translateAllProgress', { progress: updateContentProgress }) }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Right: Edit Panel -->
      <main class="flex-1">
        <div v-if="selectedLocalization" class="rounded-2xl border border-white/10 bg-white/5 p-6">
          <!-- Header -->
          <div class="flex items-center justify-between mb-6">
            <div class="flex items-center gap-3">
              <span class="text-4xl">{{ getLanguageFlag(selectedLocalization.languageCode) }}</span>
              <div>
                <h2 class="text-xl font-semibold">
                  {{ getLanguageName(selectedLocalization.languageCode) }} ({{ selectedLocalization.languageCode }})
                  <span v-if="selectedLocalization.id < 0" class="text-sm font-normal text-orange-400 ml-2">{{ t('applocalizations.add') }}</span>
                </h2>
              </div>
            </div>
            <button class="text-slate-400 hover:text-rose-500" @click="deleteLocalization(selectedLocalization.id)">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>

          <!-- Sync Actions -->
          <div v-if="selectedLocalization.id > 0" class="flex gap-2 mb-6">
            <button class="rounded-lg border border-white/20 px-3 py-2 text-sm hover:border-mint/60 hover:text-mint" @click="pullFromApple(selectedLocalization)">
              {{ t('applocalizations.pullFromApple') }}
            </button>
            <button class="rounded-lg border border-white/20 px-3 py-2 text-sm hover:border-mint/60 hover:text-mint" @click="pushToApple(selectedLocalization)">
              {{ t('applocalizations.pushToApple') }}
            </button>
          </div>

          <!-- Sync Info -->
          <div v-if="selectedLocalization.id > 0" class="flex flex-wrap gap-4 text-xs text-slate-400 mb-6 pb-4 border-b border-white/10">
            <div>{{ t('applocalizations.syncStatus') }}: {{ getSyncStatusText(selectedLocalization.syncStatus) }}</div>
            <div>{{ t('applocalizations.version') }}: {{ selectedLocalization.version || '-' }}</div>
            <div>{{ t('applocalizations.lastSynced') }}: {{ formatDate(selectedLocalization.syncedAt) }}</div>
          </div>

          <!-- Edit Form -->
          <form class="space-y-4" @submit.prevent="updateLocalization">
            <div>
              <label class="text-sm text-slate-400">{{ t('applocalizations.appName') }}</label>
              <input v-model="editLocalizationData.name" class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint" :placeholder="t('applocalizations.appNamePlaceholder')" />
            </div>

            <div>
              <label class="text-sm text-slate-400">{{ t('applocalizations.subtitle') }}</label>
              <input v-model="editLocalizationData.subtitle" class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint" :placeholder="t('applocalizations.subtitlePlaceholder')" />
            </div>

            <div>
              <label class="text-sm text-slate-400">{{ t('applocalizations.description') }}</label>
              <textarea v-model="editLocalizationData.description" rows="6" class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint whitespace-pre-wrap" :placeholder="t('applocalizations.descriptionPlaceholder')"></textarea>
            </div>

            <div>
              <label class="text-sm text-slate-400">{{ t('applocalizations.keywords') }}</label>
              <input v-model="editLocalizationData.keywords" class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint" :placeholder="t('applocalizations.keywordsPlaceholder')" />
            </div>

            <div>
              <label class="text-sm text-slate-400">{{ t('applocalizations.promotionalText') }}</label>
              <textarea v-model="editLocalizationData.promotionalText" rows="3" class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint whitespace-pre-wrap" :placeholder="t('applocalizations.promotionalTextPlaceholder')"></textarea>
            </div>

            <div>
              <label class="text-sm text-slate-400">{{ t('applocalizations.whatsNew') }}</label>
              <textarea v-model="editLocalizationData.whatsNew" rows="4" class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint whitespace-pre-wrap" :placeholder="t('applocalizations.whatsNewPlaceholder')"></textarea>
            </div>

            <div class="grid grid-cols-1 gap-4">
              <div>
                <label class="text-sm text-slate-400">{{ t('applocalizations.privacyUrl') }}</label>
                <input v-model="editLocalizationData.privacyUrl" class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint" :placeholder="t('applocalizations.privacyUrlPlaceholder')" />
              </div>
              <div>
                <label class="text-sm text-slate-400">{{ t('applocalizations.marketingUrl') }}</label>
                <input v-model="editLocalizationData.marketingUrl" class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint" :placeholder="t('applocalizations.marketingUrlPlaceholder')" />
              </div>
              <div>
                <label class="text-sm text-slate-400">{{ t('applocalizations.supportUrl') }}</label>
                <input v-model="editLocalizationData.supportUrl" class="mt-1 w-full rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint" :placeholder="t('applocalizations.supportUrlPlaceholder')" />
              </div>
            </div>

            <div class="flex items-center justify-end gap-2 pt-4">
              <select
                v-model="selectedTranslateProviderId"
                class="rounded-lg bg-white/5 px-3 py-2 text-sm ring-1 ring-white/10 focus:ring-2 focus:ring-mint"
                :title="t('applocalizations.translateProvider')"
              >
                <option :value="null">{{ t('applocalizations.localHunyuan') }}</option>
                <option v-for="config in translationProviderConfigs" :key="config.id" :value="config.id">
                  {{ getProviderDisplayName(config.providerType) }}{{ config.isDefault ? ' ★' : '' }}
                </option>
              </select>
              <button
                type="button"
                class="rounded-lg border border-white/20 px-4 py-2 text-sm hover:border-mint/60 hover:text-mint"
                @click="translateLocalization"
                :disabled="translating"
              >
                {{ translating ? t('common.translating') : t('common.translate') }}
              </button>
              <button type="submit" class="rounded-lg bg-mint px-4 py-2 text-sm font-semibold text-midnight shadow hover:bg-mint/90">
                {{ t('common.save') }}
              </button>
            </div>
          </form>
        </div>

        <!-- Empty State when no language selected -->
        <div v-else-if="localizations.length === 0" class="rounded-2xl border border-white/10 bg-white/5 p-12 text-center">
          <div class="text-6xl mb-4">🌐</div>
          <h3 class="text-lg font-semibold mb-2">{{ t('applocalizations.noLocalizations') }}</h3>
          <p class="text-sm text-slate-400">{{ t('applocalizations.noLocalizationsDesc') }}</p>
        </div>

        <!-- Select a language prompt -->
        <div v-else class="rounded-2xl border border-white/10 bg-white/5 p-12 text-center">
          <div class="text-6xl mb-4">👈</div>
          <h3 class="text-lg font-semibold mb-2">{{ t('applocalizations.selectLanguage') }}</h3>
          <p class="text-sm text-slate-400">{{ t('applocalizations.selectLanguageDesc') }}</p>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useApi } from '../composables/useApi'
import { useToast } from '../composables/useToast'
import type { AppLocalization, ProviderConfig } from '../composables/useApi'

const { t } = useI18n()
const route = useRoute()
const { api } = useApi()
const toast = useToast()

const appId = computed(() => Number(route.params.id))
const hasValidAppId = computed(() => Number.isFinite(appId.value) && appId.value > 0)

const localizations = ref<AppLocalization[]>([])
const app = ref<{ id: number; primaryLocale: string } | null>(null)
const selectedLocalizationId = ref<number | null>(null)
const selectedLanguageToAdd = ref<string>('')
const isNewLocalization = ref(false)
const showSyncModal = ref(false)
const syncing = ref(false)
const syncResult = ref<{ message: string; count?: number; conflicts?: any[] } | null>(null)
const selectedConfigId = ref<number | null>(null)
const syncDirection = ref('pull')
const syncStrategy = ref('apple_first')
const selectedSyncLanguages = ref<string[]>([])
const appleConnectConfigs = ref<ProviderConfig[]>([])
const validationErrors = ref<Record<string, string>>({})
const availableLanguages = ref<{ code: string; name: string; native_name: string; region?: string; direction: string; emoji: string, }[]>([])
const loadingLanguages = ref(false)
const translating = ref(false)

// Batch translate state
const showBatchTranslateModal = ref(false)
const selectedBatchTranslateLanguages = ref<string[]>([])
const batchTranslating = ref(false)
const batchTranslateProgress = ref(0)
const batchTranslateResult = ref<{ message: string; progress?: number; done?: number; total?: number } | null>(null)
const overwriteExisting = ref(false) // Whether to overwrite existing translations

// Translation provider selection (shared by single / batch / update-content flows).
// A null selection means the local Hunyuan (llama) model configured on the backend.
const translationProviderConfigs = ref<ProviderConfig[]>([])
const selectedTranslateProviderId = ref<number | null>(null)

// translateProviderPayload returns the providerType / providerConfigId to send to the backend
// based on the currently selected translation provider.
function translateProviderPayload(): { providerType: string; providerConfigId?: number } {
  if (selectedTranslateProviderId.value != null) {
    const cfg = translationProviderConfigs.value.find(c => c.id === selectedTranslateProviderId.value)
    return { providerType: cfg?.providerType || 'llama', providerConfigId: selectedTranslateProviderId.value }
  }
  return { providerType: 'llama' }
}

// Update Content (What's New + Promotional Text) translate state
const showUpdateContentModal = ref(false)
const selectedUpdateContentLanguages = ref<string[]>([])
const updateContentTranslating = ref(false)
const updateContentProgress = ref(0)
const updateContentResult = ref<{ message: string; progress?: number; done?: number; total?: number } | null>(null)

// Computed property for selected localization
const selectedLocalization = computed(() => {
  return localizations.value.find(loc => loc.id === selectedLocalizationId.value) || null
})

async function fetchLanguages() {
  loadingLanguages.value = true
  try {
    // Fetch Apple Connect supported languages from API
    const response = await api.getSupportedLanguages()
    if (response.success) {
      availableLanguages.value = response.languages.map(lang => ({
        code: lang.code,
        name: lang.name,
        native_name: lang.native_name,
        direction: lang.direction,
        emoji: lang.emoji || '🌐'
      }))
    }
  } catch (error) {
    console.error('Failed to load languages:', error)
  } finally {
    loadingLanguages.value = false
  }
}

function getLanguageFlag(code: string): string {
  // Find in availableLanguages
  const lang = availableLanguages.value.find(l => l.code === code)
  if (lang && lang.emoji) {
    return lang.emoji
  }
  
  // Try to extract the base language code (e.g., 'en-US' -> 'en')
  const baseCode = code.split('-')[0]
  const baseLang = availableLanguages.value.find(l => l.code === baseCode)
  if (baseLang && baseLang.emoji) {
    return baseLang.emoji
  }
  
  // Default flag for unknown languages
  return '🌐'
}

const editLocalizationData = reactive({
  id: 0,
  languageCode: '',
  name: '',
  subtitle: '',
  privacyUrl: '',
  marketingUrl: '',
  supportUrl: '',
  shortDescription: '',
  description: '',
  keywords: '',
  whatsNew: '',
  promotionalText: ''
})

// Validation rules based on Apple App Store Connect requirements
const validationRules = {
  name: { maxLength: 30, required: true },
  subtitle: { maxLength: 30, required: false },
  description: { maxLength: 4000, required: false },
  keywords: { maxLength: 100, required: false },
  privacyUrl: { maxLength: 255, required: false },
  marketingUrl: { maxLength: 255, required: false },
  supportUrl: { maxLength: 255, required: false },
  promotionalText: { maxLength: 170, required: false },
  whatsNew: { maxLength: 4000, required: false }
}

function validateField(field: string, value: string): string | null {
  const rule = validationRules[field as keyof typeof validationRules]
  if (!rule) return null

  if (rule.required && !value.trim()) {
    return 'This field is required'
  }

  if (rule.maxLength && value.length > rule.maxLength) {
    return `Must not exceed ${rule.maxLength} characters`
  }

  return null
}

function validateLocalization(data: typeof editLocalizationData): boolean {
  const errors: Record<string, string> = {}
  let isValid = true

  for (const [field, value] of Object.entries(data)) {
    const error = validateField(field, value as string)
    if (error) {
      errors[field] = error
      isValid = false
    }
  }

  validationErrors.value = errors
  return isValid
}

function getFieldError(field: string): string | null {
  return validationErrors.value[field] || null
}

function clearValidationErrors() {
  validationErrors.value = {}
}

function getLanguageName(code: string): string {
  const lang = availableLanguages.value.find(l => l.code === code)
  return lang ? lang.name : code
}

// Computed property for select all sync languages
const selectAllSyncLanguages = computed(() => {
  if (availableLanguages.value.length === 0) {
    return false
  }
  return availableLanguages.value.length > 0 &&
    availableLanguages.value.every(lang => selectedSyncLanguages.value.includes(lang.code))
})

// Toggle select all sync languages
function toggleSelectAllSyncLanguages() {
  const allSelected = selectAllSyncLanguages.value

  if (allSelected) {
    // Deselect all
    selectedSyncLanguages.value = []
  } else {
    // Select all
    selectedSyncLanguages.value = availableLanguages.value.map(lang => lang.code)
  }
}

function closeSyncModal() {
  showSyncModal.value = false
  selectedConfigId.value = null
  syncDirection.value = 'pull'
  syncStrategy.value = 'apple_first'
  selectedSyncLanguages.value = []
  syncResult.value = null
}

// Watch for selection changes to populate edit form
watch(selectedLocalizationId, (newId) => {
  if (newId) {
    const loc = localizations.value.find(l => l.id === newId)
    if (loc) {
      editLocalization(loc)
    }
  }
})

function getProviderDisplayName(providerType: string): string {
  const names: Record<string, string> = {
    'openai': 'OpenAI',
    'google': 'Google',
    'deepl': 'DeepL',
    'baidu': 'Baidu',
    'appleconnect': 'Apple Connect',
    'llama': 'Hunyuan (Llama)'
  }
  return names[providerType] || providerType
}

function getSyncStatusText(status?: string): string {
  if (!status) return 'Unknown';
  const statusMap: Record<string, string> = {
    'synced': 'Synced',
    'pending': 'Pending',
    'failed': 'Failed'
  };
  return statusMap[status] || status || 'Unknown';
}

function getSourceText(source?: string): string {
  if (!source) return 'Unknown';
  const sourceMap: Record<string, string> = {
    'apple': 'Apple',
    'local': 'Local'
  };
  return sourceMap[source] || source || 'Unknown';
}

function getVersionStateText(state?: string): string {
  if (!state) return 'Unknown';
  const stateMap: Record<string, string> = {
    'PREPARE_FOR_SUBMISSION': 'Prepare for Submission',
    'WAITING_FOR_REVIEW': 'Waiting for Review',
    'IN_REVIEW': 'In Review',
    'PENDING_DEVELOPER_RELEASE': 'Pending Developer Release',
    'PENDING_APPLE_RELEASE': 'Pending Apple Release',
    'READY_FOR_SALE': 'Ready for Sale',
    'REJECTED': 'Rejected',
    'REMOVED_FROM_SALE': 'Removed from Sale',
    'DEVELOPER_REJECTED': 'Developer Rejected',
    'METADATA_REJECTED': 'Metadata Rejected'
  };
  return stateMap[state] || state || 'Unknown';
}

function isLocalizationEditable(loc: AppLocalization): boolean {
  // Only allow editing if:
  // 1. Source is local (manually created)
  // 2. Or source is apple but version state is not READY_FOR_SALE
  if (loc.source === 'local') return true;
  if (loc.source === 'apple') {
    return loc.versionState !== 'READY_FOR_SALE';
  }
  return true;
}

function formatDate(dateString?: string): string {
  if (!dateString) return 'N/A';
  const date = new Date(dateString);
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

async function pullFromApple(localization: AppLocalization) {
  if (!confirm(`Pull localization for ${localization.languageCode} from Apple?`)) {
    return;
  }
  
  // Auto-select config if none is selected
  if (!selectedConfigId.value && appleConnectConfigs.value.length > 0) {
    const defaultConfig = appleConnectConfigs.value.find(c => c.isDefault)
    selectedConfigId.value = defaultConfig ? defaultConfig.id : appleConnectConfigs.value[0].id
  }
  
  if (!selectedConfigId.value || !hasValidAppId.value) {
    toast.warning('Please select an Apple Connect configuration first.')
    showSyncModal.value = true
    return;
  }

  try {
    const response = await api.syncAppleAppLocalizations(appId.value, {
      configId: selectedConfigId.value
    });

    if (response.success) {
      await fetchLocalizations();
      toast.success(`Successfully pulled localization for ${localization.languageCode} from Apple.`)
    } else {
      toast.error(`Failed to pull localization: ${response.message || 'Unknown error'}`)
    }
  } catch (error) {
    console.error('Failed to pull from Apple:', error);
    toast.error('Failed to pull localization from Apple. See console for details.')
  }
}

async function pushToApple(localization: AppLocalization) {
  if (!confirm(`Push localization for ${localization.languageCode} to Apple?`)) {
    return;
  }
  
  // Auto-select config if none is selected
  if (!selectedConfigId.value && appleConnectConfigs.value.length > 0) {
    const defaultConfig = appleConnectConfigs.value.find(c => c.isDefault)
    selectedConfigId.value = defaultConfig ? defaultConfig.id : appleConnectConfigs.value[0].id
  }
  
  if (!selectedConfigId.value || !hasValidAppId.value) {
    toast.warning('Please select an Apple Connect configuration first.')
    showSyncModal.value = true
    return;
  }

  try {
    const response = await api.syncAppToApple(appId.value, {
      configId: selectedConfigId.value,
      languageCode: localization.languageCode
    });

    if (response.success) {
      await fetchLocalizations();
      toast.success(`Successfully pushed localization for ${localization.languageCode} to Apple.`)
    } else {
      toast.error(`Failed to push localization: ${response.message || 'Unknown error'}`)
    }
  } catch (error) {
    console.error('Failed to push to Apple:', error);
    toast.error('Failed to push localization to Apple. See console for details.')
  }
}

async function fetchProviderConfigs() {
  try {
    // Fetch Apple Connect configs
    const appleResponse = await api.getAppleConnectConfigs()
    if (appleResponse.success) {
      appleConnectConfigs.value = appleResponse.data || []

      // Auto-select the first config if none is selected
      if (appleConnectConfigs.value.length > 0 && !selectedConfigId.value) {
        // Check if there's a default config (isDefault = true)
        const defaultConfig = appleConnectConfigs.value.find(c => c.isDefault)
        if (defaultConfig) {
          selectedConfigId.value = defaultConfig.id
        } else {
          // If no default config, select the first one
          selectedConfigId.value = appleConnectConfigs.value[0].id
        }
      }
    }

    // Fetch translation provider configs (everything except Apple Connect bindings)
    const providerResponse = await api.getProviderConfigs()
    if (providerResponse.success) {
      translationProviderConfigs.value = providerResponse.configs.filter(c => c.providerType !== 'appleconnect')

      // Default to the user's default translation provider if one is set; otherwise keep the
      // local Hunyuan model selected.
      const defaultProvider = translationProviderConfigs.value.find(c => c.isDefault)
      if (defaultProvider) {
        selectedTranslateProviderId.value = defaultProvider.id
      }
    }
  } catch (error) {
    console.error('Failed to fetch configs:', error)
  }
}

async function syncLocalizations() {
  if (!selectedConfigId.value || !hasValidAppId.value) return

  syncing.value = true
  syncResult.value = null

  try {
    // Get the selected config to extract credentials
    const selectedConfig = appleConnectConfigs.value.find(config => config.id === selectedConfigId.value);
    if (!selectedConfig) {
      throw new Error('Selected configuration not found');
    }

    const response = await api.syncAppleAppLocalizations(appId.value, {
      configId: selectedConfigId.value,
      languageCodes: selectedSyncLanguages.value.length > 0 ? selectedSyncLanguages.value : undefined,
      direction: syncDirection.value,
      strategy: syncStrategy.value
    })

    if (response.success) {
      syncResult.value = {
        message: response.message || t('applocalizations.syncSuccess'),
        count: response.count,
        conflicts: response.conflicts
      }
      // Refresh localizations list
      await fetchLocalizations()
    } else {
      syncResult.value = {
        message: response.message || t('applocalizations.syncFailed')
      }
    }
  } catch (error) {
    console.error('Failed to sync localizations:', error)
    syncResult.value = {
      message: t('applocalizations.syncError')
    }
  } finally {
    syncing.value = false
  }
}

async function updateLocalization() {
  if (!validateLocalization(editLocalizationData)) {
    return
  }

  try {
    if (!hasValidAppId.value) {
      throw new Error('Invalid app ID')
    }

    let response
    if (isNewLocalization.value) {
      // Create new localization
      response = await api.createAppLocalization(appId.value, {
        languageCode: editLocalizationData.languageCode,
        name: editLocalizationData.name,
        subtitle: editLocalizationData.subtitle,
        privacyUrl: editLocalizationData.privacyUrl,
        marketingUrl: editLocalizationData.marketingUrl,
        supportUrl: editLocalizationData.supportUrl,
        downloadDescription: '',
        shortDescription: editLocalizationData.shortDescription,
        description: editLocalizationData.description,
        keywords: editLocalizationData.keywords,
        whatsNew: editLocalizationData.whatsNew,
        promotionalText: editLocalizationData.promotionalText
      })
    } else {
      // Update existing localization
      response = await api.updateAppLocalization(appId.value, editLocalizationData.languageCode, {
        name: editLocalizationData.name,
        subtitle: editLocalizationData.subtitle,
        privacyUrl: editLocalizationData.privacyUrl,
        marketingUrl: editLocalizationData.marketingUrl,
        supportUrl: editLocalizationData.supportUrl,
        shortDescription: editLocalizationData.shortDescription,
        description: editLocalizationData.description,
        keywords: editLocalizationData.keywords,
        whatsNew: editLocalizationData.whatsNew,
        promotionalText: editLocalizationData.promotionalText
      })
    }

    if (response.success) {
      isNewLocalization.value = false
      await fetchLocalizations()
      // Select the newly created/updated localization
      if (response.localization) {
        selectedLocalizationId.value = response.localization.id
      }
      toast.success(isNewLocalization.value ? 'Localization added successfully!' : 'Localization updated successfully!')
    } else {
      toast.error(response.message || (isNewLocalization.value ? 'Failed to add localization' : 'Failed to update localization'))
    }
  } catch (error) {
    console.error('Failed to save localization:', error)
    toast.error(isNewLocalization.value ? 'Failed to add localization. Please try again.' : 'Failed to update localization. Please try again.')
  }
}

async function fetchApp() {
  if (!hasValidAppId.value) return
  try {
    const response = await api.getApp(appId.value)
    if (response.success) {
      app.value = response.app
    }
  } catch (error) {
    console.error('Failed to fetch app:', error)
  }
}

async function fetchLocalizations() {
  if (!hasValidAppId.value) return
  try {
    const response = await api.getAppLocalizations(appId.value)
    if (response.success) {
      // 去重：按 languageCode 去重，保留 ID 最大的（最新的记录）
      const seen = new Map<string, AppLocalization>()
      for (const loc of response.localizations) {
        const existing = seen.get(loc.languageCode)
        if (!existing || loc.id > existing.id) {
          seen.set(loc.languageCode, loc)
        }
      }
      localizations.value = Array.from(seen.values())
    }
  } catch (error) {
    console.error('Failed to fetch localizations:', error)
  }
}

async function deleteLocalization(id: number) {
  const localization = localizations.value.find(loc => loc.id === id)
  
  // Check if it's the primary language
  if (localization && app.value?.primaryLocale === localization.languageCode) {
    toast.warning('Cannot delete the primary language.')
    return
  }
  
  if (!confirm(t('applocalizations.confirmDelete'))) {
    return
  }
  
  // If it's a new unsaved localization (negative ID), just remove from list
  if (id < 0) {
    localizations.value = localizations.value.filter(loc => loc.id !== id)
    selectedLocalizationId.value = null
    isNewLocalization.value = false
    return
  }
  
  // For existing localizations, delete via API
  try {
    if (!hasValidAppId.value) {
      throw new Error('Invalid app ID')
    }
    const languageCode = localizations.value.find(loc => loc.id === id)?.languageCode
    
    if (languageCode) {
      await api.deleteAppLocalization(appId.value, languageCode)
      localizations.value = localizations.value.filter(loc => loc.id !== id)
    }
  } catch (error) {
    console.error('Failed to delete localization:', error)
  }
}

async function translateLocalization() {
  if (!editLocalizationData.languageCode) {
    toast.warning('Please select a language first.')
    return
  }

  translating.value = true

  try {
    // Get source language (en-US or primary locale)
    const sourceLanguage = app.value?.primaryLocale || 'en-US'
    const targetLanguage = editLocalizationData.languageCode
    // null selection => local Hunyuan; otherwise translate via the saved provider config
    const providerConfigId = selectedTranslateProviderId.value ?? undefined

    // Translate name
    if (editLocalizationData.name) {
      const nameResult = await api.translateText(editLocalizationData.name, sourceLanguage, targetLanguage, providerConfigId)
      if (nameResult.success) {
        editLocalizationData.name = nameResult.text
      }
    }

    // Translate subtitle
    if (editLocalizationData.subtitle) {
      const subtitleResult = await api.translateText(editLocalizationData.subtitle, sourceLanguage, targetLanguage, providerConfigId)
      if (subtitleResult.success) {
        editLocalizationData.subtitle = subtitleResult.text
      }
    }

    // Translate long description
    if (editLocalizationData.description) {
      const descResult = await api.translateText(editLocalizationData.description, sourceLanguage, targetLanguage, providerConfigId)
      if (descResult.success) {
        editLocalizationData.description = descResult.text
      }
    }

    // Translate keywords
    if (editLocalizationData.keywords) {
      const keywordsResult = await api.translateText(editLocalizationData.keywords, sourceLanguage, targetLanguage, providerConfigId)
      if (keywordsResult.success) {
        editLocalizationData.keywords = keywordsResult.text
      }
    }

    // Translate promotional text
    if (editLocalizationData.promotionalText) {
      const promoResult = await api.translateText(editLocalizationData.promotionalText, sourceLanguage, targetLanguage, providerConfigId)
      if (promoResult.success) {
        editLocalizationData.promotionalText = promoResult.text
      }
    }

    // Translate what's new
    if (editLocalizationData.whatsNew) {
      const notesResult = await api.translateText(editLocalizationData.whatsNew, sourceLanguage, targetLanguage, providerConfigId)
      if (notesResult.success) {
        editLocalizationData.whatsNew = notesResult.text
      }
    }

    toast.success('Translation completed successfully!')
  } catch (error) {
    console.error('Translation failed:', error)
    toast.error('Translation failed. Please try again.')
  } finally {
    translating.value = false
  }
}

// Computed property for select all languages
const selectAllLanguages = computed(() => {
  if (!app.value || availableLanguages.value.length === 0) {
    return false
  }
  const selectableLanguages = availableLanguages.value.filter(lang => lang.code !== app.value?.primaryLocale)
  return selectableLanguages.length > 0 &&
    selectableLanguages.every(lang => selectedBatchTranslateLanguages.value.includes(lang.code))
})

// Computed property for select all update content languages
const selectAllUpdateLanguages = computed(() => {
  if (!app.value || availableLanguages.value.length === 0) {
    return false
  }
  const selectableLanguages = availableLanguages.value.filter(lang => lang.code !== app.value?.primaryLocale)
  return selectableLanguages.length > 0 &&
    selectableLanguages.every(lang => selectedUpdateContentLanguages.value.includes(lang.code))
})

// Toggle select all languages
function toggleSelectAllLanguages() {
  if (!app.value) {
    return
  }

  const selectableLanguages = availableLanguages.value.filter(lang => lang.code !== app.value?.primaryLocale)
  const allSelected = selectAllLanguages.value

  if (allSelected) {
    // Deselect all
    selectedBatchTranslateLanguages.value = []
  } else {
    // Select all
    selectedBatchTranslateLanguages.value = selectableLanguages.map(lang => lang.code)
  }
}

// Toggle select all update content languages
function toggleSelectAllUpdateLanguages() {
  if (!app.value) {
    return
  }

  const selectableLanguages = availableLanguages.value.filter(lang => lang.code !== app.value?.primaryLocale)
  const allSelected = selectAllUpdateLanguages.value

  if (allSelected) {
    // Deselect all
    selectedUpdateContentLanguages.value = []
  } else {
    // Select all
    selectedUpdateContentLanguages.value = selectableLanguages.map(lang => lang.code)
  }
}

function closeBatchTranslateModal() {
  showBatchTranslateModal.value = false
  selectedBatchTranslateLanguages.value = []
  batchTranslateResult.value = null
  batchTranslateProgress.value = 0
  overwriteExisting.value = false
}

function closeUpdateContentModal() {
  showUpdateContentModal.value = false
  selectedUpdateContentLanguages.value = []
  updateContentResult.value = null
  updateContentProgress.value = 0
}

async function submitUpdateContentTranslate() {
  if (!hasValidAppId.value) {
    toast.warning('Invalid app ID')
    return
  }

  if (selectedUpdateContentLanguages.value.length === 0) {
    toast.warning('Please select at least one target language.')
    return
  }

  updateContentTranslating.value = true
  updateContentProgress.value = 0
  updateContentResult.value = { message: t('applocalizations.translateAllSubmitted'), progress: 0 }

  try {
    // Get source language (primary locale or en-US)
    const sourceLanguage = app.value?.primaryLocale || 'en-US'

    // Submit translation job with onlyTranslateWhatsNew=true (includes PromotionalText too)
    const { providerType, providerConfigId } = translateProviderPayload()
    const configData: Record<string, any> = {}
    if (providerType === 'llama') {
      Object.assign(configData, { threads: 4, temperature: 0.7, topP: 0.6, topK: 20, tokens: 4096 })
    }
    const response = await api.translateAppLocalizations(appId.value, {
      providerType,
      providerConfigId,
      sourceLanguage: sourceLanguage,
      targetLanguages: selectedUpdateContentLanguages.value,
      onlyTranslateWhatsNew: true, // This now includes both What's New and Promotional Text
      configData
    })

    if (response.success) {
      updateContentResult.value = {
        message: t('applocalizations.translateAllSuccess'),
        progress: 0
      }

      // Start polling for progress
      pollUpdateContentProgress(response.job.id)

      toast.success(t('applocalizations.translateAllSuccess'))
    } else {
      throw new Error(response.message || t('applocalizations.translateAllFailed'))
    }
  } catch (error) {
    console.error('Failed to submit update content translation:', error)
    updateContentResult.value = {
      message: t('applocalizations.translateAllError', { error: error instanceof Error ? error.message : 'Unknown error' })
    }
    toast.error(t('applocalizations.translateAllFailed'))
    updateContentTranslating.value = false
  }
}

async function pollUpdateContentProgress(jobId: number) {
  const pollInterval = setInterval(async () => {
    try {
      const response = await api.getQueueJob(jobId)
      if (response.success) {
        const job = response.job

        // Update progress
        updateContentProgress.value = job.progress || 0
        updateContentResult.value = {
          message: t('applocalizations.translateAllSubmitted'),
          progress: updateContentProgress.value,
          done: job.done,
          total: job.total
        }

        // Check if job is completed
        if (job.status === 'completed') {
          clearInterval(pollInterval)
          updateContentTranslating.value = false
          updateContentResult.value = {
            message: t('applocalizations.translateAllCompleted'),
            progress: 100,
            done: job.done,
            total: job.total
          }

          // Refresh localizations
          await fetchLocalizations()
          toast.success(t('applocalizations.translateAllCompleted'))
        } else if (job.status === 'failed') {
          clearInterval(pollInterval)
          updateContentTranslating.value = false
          updateContentResult.value = {
            message: t('applocalizations.translateAllError', { error: job.error || 'Unknown error' }),
            done: job.done,
            total: job.total
          }
          toast.error(t('applocalizations.translateAllError', { error: job.error || 'Unknown error' }))
        }
      }
    } catch (error) {
      console.error('Failed to poll translation progress:', error)
      clearInterval(pollInterval)
      updateContentTranslating.value = false
    }
  }, 2000) // Poll every 2 seconds
}

async function submitBatchTranslate() {
  if (!hasValidAppId.value) {
    toast.warning('Invalid app ID')
    return
  }

  if (selectedBatchTranslateLanguages.value.length === 0) {
    toast.warning('Please select at least one target language.')
    return
  }

  batchTranslating.value = true
  batchTranslateProgress.value = 0
  batchTranslateResult.value = { message: t('applocalizations.translateAllSubmitted'), progress: 0 }

  try {
    // Get source language (primary locale or en-US)
    const sourceLanguage = app.value?.primaryLocale || 'en-US'

    // Submit translation job using the selected provider (local Hunyuan or a saved provider
    // such as an OpenAI-compatible endpoint). When llama is used the backend falls back to the
    // default configuration from backend/config.yaml. For saved providers the backend resolves
    // the real config (including secrets) from providerConfigId.
    const { providerType, providerConfigId } = translateProviderPayload()
    const configData: Record<string, any> = {
      // Skip existing translations if overwrite is false
      skipExisting: !overwriteExisting.value
    }
    if (providerType === 'llama') {
      // Hunyuan model generation parameters
      Object.assign(configData, { threads: 4, temperature: 0.7, topP: 0.6, topK: 20, tokens: 4096 })
    }
    const response = await api.translateAppLocalizations(appId.value, {
      providerType,
      providerConfigId,
      sourceLanguage: sourceLanguage,
      targetLanguages: selectedBatchTranslateLanguages.value,
      configData
    })

    if (response.success) {
      batchTranslateResult.value = {
        message: t('applocalizations.translateAllSuccess'),
        progress: 0
      }

      // Start polling for progress
      pollTranslationProgress(response.job.id)

      toast.success(t('applocalizations.translateAllSuccess'))
    } else {
      throw new Error(response.message || t('applocalizations.translateAllFailed'))
    }
  } catch (error) {
    console.error('Failed to submit batch translation:', error)
    batchTranslateResult.value = {
      message: t('applocalizations.translateAllError', { error: error instanceof Error ? error.message : 'Unknown error' })
    }
    toast.error(t('applocalizations.translateAllFailed'))
    batchTranslating.value = false
  }
}

async function pollTranslationProgress(jobId: number) {
  const pollInterval = setInterval(async () => {
    try {
      const response = await api.getQueueJob(jobId)
      if (response.success) {
        const job = response.job

        // Update progress
        batchTranslateProgress.value = job.progress || 0
        batchTranslateResult.value = {
          message: t('applocalizations.translateAllSubmitted'),
          progress: batchTranslateProgress.value,
          done: job.done,
          total: job.total
        }

        // Check if job is completed
        if (job.status === 'completed') {
          clearInterval(pollInterval)
          batchTranslating.value = false
          batchTranslateResult.value = {
            message: t('applocalizations.translateAllCompleted'),
            progress: 100,
            done: job.done,
            total: job.total
          }

          // Refresh localizations
          await fetchLocalizations()
          toast.success(t('applocalizations.translateAllCompleted'))
        } else if (job.status === 'failed') {
          clearInterval(pollInterval)
          batchTranslating.value = false
          batchTranslateResult.value = {
            message: t('applocalizations.translateAllError', { error: job.error || 'Unknown error' }),
            done: job.done,
            total: job.total
          }
          toast.error(t('applocalizations.translateAllError', { error: job.error || 'Unknown error' }))
        }
      }
    } catch (error) {
      console.error('Failed to poll translation progress:', error)
      clearInterval(pollInterval)
      batchTranslating.value = false
    }
  }, 2000) // Poll every 2 seconds
}

function editLocalization(localization: AppLocalization) {
  // Populate edit form with existing data
  editLocalizationData.id = localization.id
  editLocalizationData.languageCode = localization.languageCode
  editLocalizationData.name = localization.name || ''
  editLocalizationData.subtitle = localization.subtitle || ''
  editLocalizationData.privacyUrl = localization.privacyUrl || ''
  editLocalizationData.marketingUrl = localization.marketingUrl || ''
  editLocalizationData.supportUrl = localization.supportUrl || ''
  editLocalizationData.shortDescription = localization.shortDescription || ''
  editLocalizationData.description = localization.description || localization.description || ''
  editLocalizationData.keywords = localization.keywords || ''
  editLocalizationData.whatsNew = localization.whatsNew || ''
  editLocalizationData.promotionalText = localization.promotionalText || ''

  clearValidationErrors()
}

async function handleLanguageSelect() {
  if (!selectedLanguageToAdd.value) return

  // Check if language already exists
  const existing = localizations.value.find(loc => loc.languageCode === selectedLanguageToAdd.value)
  if (existing) {
    // Select existing language
    selectedLocalizationId.value = existing.id
    selectedLanguageToAdd.value = ''
    return
  }

  // Get en-US localization data for default values
  const enUsLocalization = localizations.value.find(loc => loc.languageCode === 'en-US' || loc.languageCode === 'en')

  // Prepare new localization form with en-US data
  editLocalizationData.id = 0
  editLocalizationData.languageCode = selectedLanguageToAdd.value
  editLocalizationData.name = enUsLocalization?.name || ''
  editLocalizationData.subtitle = enUsLocalization?.subtitle || ''
  editLocalizationData.privacyUrl = enUsLocalization?.privacyUrl || ''
  editLocalizationData.marketingUrl = enUsLocalization?.marketingUrl || ''
  editLocalizationData.supportUrl = enUsLocalization?.supportUrl || ''
  editLocalizationData.shortDescription = enUsLocalization?.shortDescription || ''
  editLocalizationData.description = enUsLocalization?.description || enUsLocalization?.description || ''
  editLocalizationData.keywords = enUsLocalization?.keywords || ''
  editLocalizationData.whatsNew = enUsLocalization?.whatsNew || ''
  editLocalizationData.promotionalText = enUsLocalization?.promotionalText || ''

  clearValidationErrors()
  isNewLocalization.value = true

  // Add temporary entry to list
  const tempId = -Date.now()
  const tempLocalization: AppLocalization = {
    id: tempId,
    appId: appId.value,
    languageCode: selectedLanguageToAdd.value,
    name: editLocalizationData.name,
    subtitle: editLocalizationData.subtitle,
    description: editLocalizationData.description,
    keywords: editLocalizationData.keywords,
    privacyUrl: editLocalizationData.privacyUrl,
    marketingUrl: editLocalizationData.marketingUrl,
    supportUrl: editLocalizationData.supportUrl,
    promotionalText: editLocalizationData.promotionalText,
    whatsNew: editLocalizationData.whatsNew,
    syncStatus: 'pending',
    version: '',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    versionState: '',
    syncedAt: '',
    source: 'local'
  }
  localizations.value.push(tempLocalization)
  selectedLocalizationId.value = tempId
  selectedLanguageToAdd.value = ''
}

watch(appId, () => {
  fetchLocalizations()
})

onMounted(() => {
  fetchApp()
  fetchLanguages()
  fetchLocalizations()
  fetchProviderConfigs()
})

// Watch for selection changes to populate edit form
watch(selectedLocalizationId, (newId) => {
  if (newId) {
    const loc = localizations.value.find(l => l.id === newId)
    if (loc) {
      editLocalization(loc)
      isNewLocalization.value = loc.id < 0
    }
  }
})
</script>
