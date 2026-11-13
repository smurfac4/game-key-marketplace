export function LanguageFlag({ language }: { language: "en" | "ru" }) {
  return (
    <svg
      role="img"
      aria-label={
        language === "ru" ? "Russian language flag" : "English language flag"
      }
      viewBox="0 0 24 16"
      className="h-4 w-6 shrink-0 rounded-sm ring-1 ring-white/25"
    >
      {language === "ru" ? (
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
