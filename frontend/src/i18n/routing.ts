import { defineRouting } from "next-intl/routing";

export const locales = ["en", "tr", "az"] as const;
export type Locale = (typeof locales)[number];

export const routing = defineRouting({
  locales,
  defaultLocale: "en",
});

export function isLocale(value: string): value is Locale {
  return locales.includes(value as Locale);
}
