import React from "react";
import { Copy, Download, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useTunnelStore } from "@/lib/tunnelStore";

interface TokenCardProps {
  token: string;
  type: "offer" | "answer";
  onExport?: () => void;
}

export const TokenCard: React.FC<TokenCardProps> = ({ token, type, onExport }) => {
  const copyToClipboard = async () => {
    await navigator.clipboard.writeText(token);
  };

  const shareLink = () => {
    const url = `minecraft-tunnel://join?token=${encodeURIComponent(token)}`;
    navigator.clipboard.writeText(url);
  };

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          {type === "offer" ? "Offer Token" : "Answer Token"}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <Input
          readOnly
          value={token}
          className="font-mono text-xs"
          placeholder="Token will appear here..."
        />
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={copyToClipboard}
            disabled={!token}
          >
            <Copy className="w-4 h-4 mr-2" />
            Copy
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={shareLink}
            disabled={!token}
          >
            <Upload className="w-4 h-4 mr-2" />
            Share Link
          </Button>
          {onExport && (
            <Button
              variant="outline"
              size="sm"
              onClick={onExport}
              disabled={!token}
            >
              <Download className="w-4 h-4 mr-2" />
              Export
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
};
