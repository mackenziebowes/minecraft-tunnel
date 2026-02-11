import React, { useEffect, useRef } from "react";
import { useTunnelStore } from "@/lib/tunnelStore";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
import Sigil from "@/components/custom/sigil";

// Icons
import { Power, ArrowLeft, Activity, Terminal } from "lucide-react";

// UI Components (ShadCN + Tailwind)
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";

export const HostView = () => {
  const {
    status,
    logs,
    ip,
    setIp,
    port,
    setPort,
    addLog,
    setStatus,
    startHost,
    stopHost,
  } = useTunnelStore();

  const scrollRef = useRef<HTMLDivElement>(null);

  // Wails Event Listeners
  useEffect(() => {
    EventsOn("log", (msg: string) => addLog(msg));
    EventsOn("status-change", (newStatus: string) =>
      setStatus(newStatus as any),
    );
    return () => {
      EventsOff("log");
      EventsOff("status-change");
    };
  }, []);

  // Auto-scroll
  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  // Dynamic Styles based on Status
  const isRunning = status === "connected";
  const statusColor =
    {
      disconnected: "border-slate-200",
      connecting: "border-yellow-200 animate-pulse",
      connected: "border-green-200",
      error: "",
    }[status] || "";

  return (
    <div className="flex flex-col items-center justify-center min-h-screen min-w-screen p-6 font-sans">
      <Sigil scale={0.25} rotating />
      <Card className="w-full shadow-xl">
        {/* Header */}
        <CardHeader className="pb-4">
          <div className="flex justify-between items-start">
            <div className="space-y-1">
              <CardTitle className="text-xl font-bold tracking-tight">
                Host Tunnel
              </CardTitle>
              <CardDescription className="">
                Expose your local Minecraft server to the world.
              </CardDescription>
            </div>
            <Badge
              variant="outline"
              className={`${statusColor} capitalize px-3 py-1`}
            >
              {status === "connected" ? (
                <Activity className="w-3 h-3 mr-1" />
              ) : null}
              {status}
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Input Section */}
          <div className="space-y-2">
            <Label htmlFor="server-ip" className="text-sm font-medium">
              Target VPS Address
            </Label>
            <div className="relative">
              <Input
                id="server-ip"
                placeholder="vps.example.com:8080"
                value={ip}
                onChange={(e) => setIp(e.target.value)}
                disabled={isRunning || status === "connecting"}
                className="font-mono text-sm pl-3 pr-10"
              />
              <Input
                id="server-port"
                placeholder="25435"
                value={port}
                onChange={(e) => setIp(e.target.value)}
                disabled={isRunning || status === "connecting"}
                className="font-mono text-sm pl-3 pr-10"
              />
              {/* distinct lock icon or status indicator inside input could go here */}
            </div>
          </div>

          {/* Terminal / Logs Section */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium flex items-center gap-2">
                <Terminal className="w-4 h-4" />
                Live Logs
              </Label>
              <span className="text-xs">
                {logs.length > 0 ? `${logs.length} events` : "Ready"}
              </span>
            </div>

            <div className="rounded-lg border p-4 shadow-inner">
              <ScrollArea className="h-64 w-full pr-4">
                <div className="flex flex-col gap-1 font-mono text-xs">
                  {logs.length === 0 && (
                    <div className="italic select-none py-10 text-center">
                      Waiting for connection...
                    </div>
                  )}
                  {logs.map((log, i) => (
                    <div
                      key={i}
                      className="break-all border-l-2 border-transparent pl-2 hover:border-slate-700 hover:bg-slate-900/50 transition-colors"
                    >
                      <span className="mr-2">
                        {new Date().toLocaleTimeString([], { hour12: false })}
                      </span>
                      {log}
                    </div>
                  ))}
                  <div ref={scrollRef} />
                </div>
              </ScrollArea>
            </div>
          </div>
        </CardContent>

        {/* Footer Actions */}
        <CardFooter className="flex justify-between pt-2 border-t border-slate-100 rounded-b-xl px-6 py-4">
          <Button
            variant="ghost"
            onClick={() => window.history.back()}
            className=""
            disabled={status !== "disconnected"}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back
          </Button>

          {status === "disconnected" || status === "error" ? (
            <Button
              onClick={startHost}
              className="shadow-md transition-all hover:shadow-lg w-32"
            >
              <Power className="w-4 h-4 mr-2" />
              Start
            </Button>
          ) : (
            <Button
              onClick={stopHost}
              variant="destructive"
              className="shadow-md transition-all hover:shadow-lg w-32"
            >
              <Power className="w-4 h-4 mr-2" />
              Stop
            </Button>
          )}
        </CardFooter>
      </Card>
    </div>
  );
};
