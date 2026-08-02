import { useEffect, useRef, useState } from "react";
import { Terminal as TerminalIcon } from "lucide-react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

interface NodeTerminalDialogProps {
  clusterId: string;
  nodeId: string;
  nodeName: string;
  disabled?: boolean;
  disabledReason?: string;
}

export function NodeTerminalDialog({
  clusterId,
  nodeId,
  nodeName,
  disabled,
  disabledReason,
}: NodeTerminalDialogProps) {
  const [open, setOpen] = useState(false);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          disabled={disabled}
          title={disabled ? disabledReason : `Open terminal on ${nodeName}`}
          aria-label={`Open terminal on ${nodeName}`}
        >
          <TerminalIcon className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Terminal — {nodeName}</DialogTitle>
          <DialogDescription>
            Interactive root shell inside the node's VM. Closing this dialog
            ends the session.
          </DialogDescription>
        </DialogHeader>
        {open && (
          <TerminalPane clusterId={clusterId} nodeId={nodeId} />
        )}
      </DialogContent>
    </Dialog>
  );
}

// Mounted only while the dialog is open, so its lifecycle (one effect, one
// cleanup) exactly matches the WebSocket + xterm session's lifetime.
function TerminalPane({
  clusterId,
  nodeId,
}: {
  clusterId: string;
  nodeId: string;
}) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const term = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontSize: 13,
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(container);
    fitAddon.fit();

    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(
      `${proto}//${window.location.host}/api/v1/clusters/${clusterId}/nodes/${nodeId}/terminal`,
    );
    ws.binaryType = "arraybuffer";

    const sendResize = () => {
      if (ws.readyState !== WebSocket.OPEN) return;
      ws.send(
        JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }),
      );
    };

    ws.onopen = () => {
      sendResize();
      term.focus();
    };
    ws.onmessage = (ev) => {
      if (typeof ev.data === "string") return; // reserved for future control frames
      term.write(new Uint8Array(ev.data as ArrayBuffer));
    };
    ws.onerror = () => {
      term.writeln("\r\n\x1b[31mConnection error.\x1b[0m");
    };
    ws.onclose = () => {
      term.writeln("\r\n\x1b[33mSession closed.\x1b[0m");
    };

    const dataDisposable = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode(data));
      }
    });

    const resizeObserver = new ResizeObserver(() => {
      fitAddon.fit();
      sendResize();
    });
    resizeObserver.observe(container);

    return () => {
      resizeObserver.disconnect();
      dataDisposable.dispose();
      ws.close();
      term.dispose();
    };
  }, [clusterId, nodeId]);

  return (
    <div
      ref={containerRef}
      className="h-[60vh] w-full overflow-hidden rounded-lg bg-black p-2"
    />
  );
}
