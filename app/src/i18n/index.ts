import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { getLocales } from 'react-native-localize';

import zhCN from './zh-CN.json';
import en from './en.json';

const resources = {
  'zh-CN': { translation: zhCN },
  en: { translation: en },
};

// Detect system language
const locales = getLocales();
const systemLang = locales[0]?.languageTag || 'en';
const defaultLang = systemLang.startsWith('zh') ? 'zh-CN' : 'en';

i18n.use(initReactI18next).init({
  resources,
  lng: defaultLang,
  fallbackLng: 'en',
  interpolation: {
    escapeValue: false,
  },
});

export default i18n;
