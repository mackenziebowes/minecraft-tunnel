import { Button, type ButtonArgs } from "../ui/button";
import { useAppStore, type ValidRoutes } from "@/lib/store";

type RouteButtonArgs = ButtonArgs & { route: ValidRoutes };

export function RouteButton({
  route,
  className,
  variant,
  size,
  children,
}: RouteButtonArgs) {
  const setRoute = useAppStore((s) => s.setRoute);
  return (
    <Button
      className={className}
      variant={variant}
      size={size}
      onClick={() => setRoute(route)}
    >
      {children}
    </Button>
  );
}
