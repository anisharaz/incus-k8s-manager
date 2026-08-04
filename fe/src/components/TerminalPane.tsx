import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

interface TerminalPaneProps {
  clusterId: string;
  nodeId: string;
  className?: string;
}

// Bridges an xterm.js instance to the node's interactive shell over the
// terminal websocket. One effect owns the whole session's lifecycle (setup
// on mount, teardown on unmount/clusterId+nodeId change) so switching nodes
// always starts from a clean connection.
export function TerminalPane({ clusterId, nodeId, className }: TerminalPaneProps) {
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
      className={className ?? "h-full w-full overflow-hidden bg-black p-2"}
    />
  );
}
