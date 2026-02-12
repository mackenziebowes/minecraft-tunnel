import React, { useEffect, useRef } from "react";
import { useTunnelStore } from "@/lib/tunnelStore";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
import { TokenCard } from "@/components/custom/token-card";
import Sigil from "@/components/custom/sigil";

import { Power, ArrowLeft, Activity, Terminal, Server } from "lucide-react";

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
    offerToken,
    mcServerAddress,
    setMcServerAddress,
    addLog,
    setStatus,
    generateOffer,
    acceptAnswer,
    exportToken,
  } = useTunnelStore();

  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    EventsOn("log", addLog);
    EventsOn("status-change", (newStatus: string) =>
      setStatus(newStatus as any),
    );
    return () => {
      EventsOff("log");
      EventsOff("status-change");
    };
  }, [addLog, setStatus]);

  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  const isRunning = status === "connected";
  const statusColor =
    {
      disconnected: "border-slate-200",
      connecting: "border-yellow-200 animate-pulse",
      "waiting-for-answer": "border-blue-200",
      "waiting-for-host": "border-blue-200",
      connected: "border-green-200",
      error: "border-red-200",
    }[status] || "";

  return (
    <div className="flex flex-col items-center justify-center min-h-screen min-w-screen p-6 font-sans">
      <Sigil scale={0.25} rotating />
      <Card className="w-full shadow-xl">
        <CardHeader className="pb-4">
          <div className="flex justify-between items-start">
            <div className="space-y-1">
              <CardTitle className="text-xl font-bold tracking-tight">
                Host Tunnel
              </CardTitle>
              <CardDescription>
                Expose your local Minecraft server to a friend.
              </CardDescription>
            </div>
            <Badge
              variant="outline"
              className={`${statusColor} capitalize px-3 py-1`}
            >
              {status === "connected" && <Activity className="w-3 h-3 mr-1" />}
              {status.replace(/-/g, " ")}
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Server Address Input */}
          <div className="space-y-2">
            <Label htmlFor="server-address" className="text-sm font-medium">
              <Server className="w-4 h-4 mr-1 inline" />
              Minecraft Server Address
            </Label>
            <Input
              id="server-address"
              placeholder="localhost:25565"
              value={mcServerAddress}
              onChange={(e) => setMcServerAddress(e.target.value)}
              disabled={isRunning}
              className="font-mono text-sm"
            />
          </div>

          {/* Offer Token Section */}
          {(status === "waiting-for-answer" || status === "connected") && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">
                Share this Offer Token with your friend:
              </Label>
              <TokenCard
                token={offerToken}
                type="offer"
                onExport={() => exportToken(offerToken)}
              />
            </div>
          )}

          {/* Answer Input Section */}
          {status === "waiting-for-answer" && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">
                Paste Answer Token from friend:
              </Label>
              <div className="flex gap-2">
                <Input
                  placeholder="Paste answer token here..."
                  className="font-mono text-xs flex-1"
                  id="answer-input"
                />
                <Button onClick={() => {
                  const input = document.getElementById("answer-input") as HTMLInputElement;
                  if (input.value) acceptAnswer(input.value);
                }}>
                  Connect
                </Button>
              </div>
            </div>
          )}

          {/* Logs */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium flex items-center gap-2">
                <Terminal className="w-4 h-4" />
                Live Logs
              </Label>
              <span className="text-xs">{logs.length} events</span>
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
                      <span className="mr-2 text-slate-500">
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

        <CardFooter className="flex justify-between pt-2 border-t border-slate-100">
          <Button
            variant="ghost"
            onClick={() => window.history.back()}
            disabled={status !== "disconnected"}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back
          </Button>

          {status === "disconnected" || status === "error" ? (
            <Button onClick={generateOffer} className="shadow-md">
              <Power className="w-4 h-4 mr-2" />
              Generate Invitation
            </Button>
          ) : (
            <Button variant="outline" disabled>
              <Activity className="w-4 h-4 mr-2" />
              {status === "connected" ? "Connected" : "Connecting..."}
            </Button>
          )}
        </CardFooter>
      </Card>
    </div>
  );
};
