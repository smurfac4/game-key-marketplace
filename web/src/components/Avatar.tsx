import { copy } from "../content/copy";
export const avatarChoices = () => [
  { value: "nebula", label: copy.avatarNebula },
  { value: "pixel", label: copy.avatarPixel },
  { value: "orbit", label: copy.avatarOrbit },
  { value: "ember", label: copy.avatarEmber },
];
export function Avatar({
  kind = "nebula",
  className = "h-11 w-11",
}: {
  kind?: string;
  className?: string;
}) {
  const colors: Record<string, string> = {
    nebula: "#b797ff",
    pixel: "#50e3c2",
    orbit: "#8bc9ff",
    ember: "#ffac71",
  };
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 64 64"
      className={`shrink-0 rounded-full border border-white/40 bg-[#24104d] ${className}`}
    >
      <circle cx="32" cy="32" r="29" fill="#ffffff" fillOpacity=".05" />
      {kind === "orbit" ? (
        <>
          <ellipse
            cx="32"
            cy="32"
            rx="25"
            ry="10"
            transform="rotate(-35 32 32)"
            fill="none"
            stroke={colors.orbit}
            strokeWidth="3"
          />
          <circle cx="32" cy="32" r="14" fill={colors.orbit} />
          <circle cx="39" cy="28" r="3" fill="#24104d" />
        </>
      ) : (
        <>
          <path
            d={
              kind === "pixel"
                ? "M15 17h34v30H15z"
                : "M16 26q0-10 16-10t16 10v19q-16 12-32 0Z"
            }
            fill={colors[kind] ?? colors.nebula}
          />
          <path
            d="M32 16V9m-3 0h6"
            stroke={colors[kind] ?? colors.nebula}
            strokeWidth="3"
            strokeLinecap="round"
          />
          <rect
            x="21"
            y="27"
            width="22"
            height="10"
            rx={kind === "pixel" ? 0 : 5}
            fill="#24104d"
          />
          <circle cx="27" cy="32" r="2" fill="white" />
          <circle cx="37" cy="32" r="2" fill="white" />
          <path
            d="M27 43h10"
            stroke="#24104d"
            strokeWidth="3"
            strokeLinecap="round"
          />
        </>
      )}
    </svg>
  );
}
