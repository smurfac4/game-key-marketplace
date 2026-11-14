import type { Preferences } from "../content/preferences";
export function LanguageFlag({
  language,
}: {
  language: Preferences["language"];
}) {
  return (
    <svg
      role="img"
      aria-label={
        {
          en: "English language flag",
          ru: "Russian language flag",
          lt: "Lithuanian language flag",
        }[language]
      }
      viewBox="0 0 24 16"
      className="h-4 w-6 shrink-0 rounded-sm ring-1 ring-white/25"
    >
      {language === "lt" ? (
        <>
          <path fill="#fdb913" d="M0 0h24v16H0z" />
          <path fill="#006a44" d="M0 5.333h24v5.334H0z" />
          <path fill="#c1272d" d="M0 10.667h24V16H0z" />
        </>
      ) : language === "ru" ? (
        <>
          <path fill="#fff" d="M0 0h24v16H0z" />
          <path fill="#0039a6" d="M0 5.333h24v5.334H0z" />
          <path fill="#d52b1e" d="M0 10.667h24V16H0z" />
        </>
      ) : (
        <>
          <path fill="#012169" d="M0 0h24v16H0z" />
          <path stroke="#fff" strokeWidth="5" d="m0 0 24 16M24 0 0 16" />
          <path stroke="#c8102e" strokeWidth="1.8" d="m0 0 24 16M24 0 0 16" />
          <path fill="#fff" d="M9.5 0h5v16h-5zM0 5.5h24v5H0z" />
          <path fill="#c8102e" d="M10.5 0h3v16h-3zM0 6.5h24v3H0z" />
        </>
      )}
    </svg>
  );
}
