import type { SVGProps } from "react";
const paths: Record<string, string> = {
  game: "M6 8h12l3 11h-4l-3-3h-4l-3 3H3L6 8Zm2 3v4m-2-2h4m6-1h.01M18 14h.01",
  globe:
    "M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0ZM3 12h18M12 3c5 5 5 13 0 18-5-5-5-13 0-18Z",
  bolt: "m13 2-9 12h7l-1 8 10-13h-8l1-7Z",
  shield: "M12 3 4 6v6c0 5 8 9 8 9s8-4 8-9V6l-8-3Zm-4 9 3 3 5-6",
  key: "M14 4a6 6 0 1 1-4 10l-7 7v-4l6-6a6 6 0 0 1 5-7Z",
  grid: "M3 3h7v7H3V3Zm11 0h7v7h-7V3ZM3 14h7v7H3v-7Zm11 0h7v7h-7v-7Z",
  arrow: "M4 12h16m-6-6 6 6-6 6",
  desktop: "M3 4h18v13H3V4Zm5 17h8m-4-4v4",
  star: "m12 2 3 6 7 1-5 5 1 7-6-3-6 3 1-7-5-5 7-1 3-6Z",
};
export function Icon({
  name = "game",
  ...props
}: SVGProps<SVGSVGElement> & {
  name?: string;
}) {
  return (
    <svg
      aria-hidden="true"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <path d={paths[name] ?? paths.game} />
    </svg>
  );
}
