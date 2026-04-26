import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import resourcesToBackend from 'i18next-resources-to-backend';

export const supportedLanguages = ['en', 'vi'] as const;
export const namespaces = ['common', 'auth', 'videos'] as const;
export const defaultNamespace = 'common';

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .use(
    resourcesToBackend(
      (language: string, namespace: string) => import(`../locales/${language}/${namespace}.json`)
    )
  )
  .init({
    fallbackLng: 'en',
    supportedLngs: supportedLanguages,
    defaultNS: defaultNamespace,
    ns: namespaces,
    fallbackNS: defaultNamespace,
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
    },
    react: { useSuspense: false },
  });

export default i18n;
