import Sigil from "@/components/custom/sigil";
import { RouteButton } from "@/components/custom/route-button";
export default function Main() {
  return (
    <div className="flex flex-col gap-2 w-full justify-center items-center">
      <Sigil scale={1} rotating />
      <div className="flex flex-col py-4">
        <h2 className="headline">Covenant</h2>
        <div className="underline" />
      </div>
      <div className="flex flex-col gap-4 mt-8">
        <div />
        <RouteButton route="/host" variant="default">
          Host
        </RouteButton>
        <RouteButton route="/join" variant="default">
          Join
        </RouteButton>
      </div>
    </div>
  );
}
