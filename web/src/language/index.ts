import { createI18n } from 'vue-i18n';
import zhNested from '../i18n/lang/zh'
import zhTw from '../i18n/lang/zh-tw'
import en from '../i18n/lang/en'
import ja from '../i18n/lang/ja'
import ko from '../i18n/lang/ko'
import vi from '../i18n/lang/vi'
import th from '../i18n/lang/th'
import id from '../i18n/lang/id'
import legacyZh from './zh'
import legacyEn from './en'
import legacyJa from './rb'
import legacyKo from './hg'
import legacyTh from './tg'
import legacyId from './xjp'
// lang('中文文案') 以中文为 key；与嵌套英文 key 并存
import zhFlat from '../i18n/language/zh.json'
import enFlat from '../i18n/language/en.json'

const initialLocale = localStorage.getItem('lan') || 'zh'
document.documentElement.lang = initialLocale

// 最少改动：把已有 flat 表注入各语言。
// 非英文语言暂用 enFlat 兜底，避免切语言后仍显示中文。
export default createI18n({
    legacy: false,
    locale: initialLocale,
    fallbackLocale: "en",
    missingWarn: false,
    fallbackWarn: false,
    messages: {
        zh: { ...legacyZh, ...zhFlat, ...zhNested },
        'zh-tw': { ...legacyZh, ...zhFlat, ...zhTw, futurefi: { ...zhNested.futurefi, ...zhTw.futurefi }, oracle: zhNested.oracle },
        en: { ...legacyEn, ...enFlat, ...en },
        ja: {
            ...legacyEn,
            ...legacyJa,
            join: { ...legacyEn.join, ...legacyJa.join },
            peopleInfo: { ...legacyEn.peopleInfo, ...legacyJa.peopleInfo },
            ...enFlat,
            ...ja,
            futurefi: { ...en.futurefi, ...ja.futurefi },
            oracle: en.oracle,
        },
        ko: {
            ...legacyEn,
            ...legacyKo,
            join: { ...legacyEn.join, ...legacyKo.join },
            peopleInfo: { ...legacyEn.peopleInfo, ...legacyKo.peopleInfo },
            ...enFlat,
            ...ko,
            futurefi: { ...en.futurefi, ...ko.futurefi },
            oracle: en.oracle,
        },
        vi: { ...legacyEn, ...enFlat, ...vi, futurefi: { ...en.futurefi, ...vi.futurefi }, oracle: en.oracle },
        th: {
            ...legacyEn,
            ...legacyTh,
            join: { ...legacyEn.join, ...legacyTh.join },
            peopleInfo: { ...legacyEn.peopleInfo, ...legacyTh.peopleInfo },
            ...enFlat,
            ...th,
            futurefi: { ...en.futurefi, ...th.futurefi },
            oracle: en.oracle,
        },
        id: {
            ...legacyEn,
            ...legacyId,
            join: { ...legacyEn.join, ...legacyId.join },
            peopleInfo: { ...legacyEn.peopleInfo, ...legacyId.peopleInfo },
            ...enFlat,
            ...id,
            futurefi: { ...en.futurefi, ...id.futurefi },
            oracle: en.oracle,
        },
    },
})
