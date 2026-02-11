interface SigilArgs {
  scale: number;
  rotating?: boolean;
}

export default function Sigil({ scale, rotating }: SigilArgs) {
  return (
    <div
      style={{
        width: `${280 * scale}px`,
        height: `${280 * scale}px`,
        transform: `scale(${scale})`,
      }}
    >
      <div className={`onboarding-circle`}>
        <div className={`central-sigil ${rotating ? "a-rotate-10" : ""}`}>
          <div className="sigil-eye"></div>
        </div>
      </div>
    </div>
  );
}
