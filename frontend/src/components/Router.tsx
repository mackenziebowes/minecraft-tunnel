import { useAppStore, type ValidRoutes } from "@/lib/store";
import Main from "@/routes/main";
import { HostView } from "@/routes/host";

export function Router() {
  const route = useAppStore((s) => s.route);
  switch (route) {
    case "/":
      return (
        <>
          <Main />
        </>
      );
    case "/host":
      return (
        <>
          <HostView />
        </>
      );
    case "/join":
      return <></>;
  }
}
