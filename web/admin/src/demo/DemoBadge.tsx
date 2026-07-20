import { resetState } from "./state";

// Fixed badge shown only in demo builds: says where the data lives and lets
// the visitor start the tour over. Styled inline so demo mode adds nothing
// to the production stylesheet.
export function DemoBadge() {
  return (
    <div
      style={{
        position: "fixed",
        bottom: 16,
        right: 16,
        zIndex: 1000,
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "8px 12px",
        borderRadius: 999,
        background: "rgba(20, 22, 30, 0.92)",
        color: "rgba(255, 255, 255, 0.92)",
        font: "500 12px/1.4 system-ui, sans-serif",
        boxShadow: "0 4px 16px rgba(0, 0, 0, 0.25)",
      }}
    >
      <span>Demo — data stays in your browser</span>
      <button
        type="button"
        onClick={() => {
          resetState();
          window.location.reload();
        }}
        style={{
          border: "1px solid rgba(255, 255, 255, 0.3)",
          borderRadius: 999,
          background: "transparent",
          color: "inherit",
          font: "inherit",
          padding: "2px 10px",
          cursor: "pointer",
        }}
      >
        Reset
      </button>
    </div>
  );
}
