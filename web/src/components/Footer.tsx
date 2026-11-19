import { useTranslation } from "react-i18next";

export function Footer() {
  const { t } = useTranslation();
  return (
    <footer className="flex flex-wrap items-center justify-between gap-2 bg-white/60 px-6 py-5 text-sm text-muted">
      <p>{t("footer.rights", { year: new Date().getFullYear() })}</p>
      <p>♡ {t("footer.love")}</p>
    </footer>
  );
}
