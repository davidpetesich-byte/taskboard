import type { Ticket } from "../api/client";

export function ticketKey(ticket: Pick<Ticket, "projectPrefix" | "number">): string {
  return `${ticket.projectPrefix}-${ticket.number}`;
}

// A board link opens the ticket on the board. Claude Code sessions read the
// same key through the taskboard MCP get_ticket tool.
export function ticketLink(ticket: Pick<Ticket, "projectPrefix" | "number">): string {
  return `${window.location.origin}/?ticket=${encodeURIComponent(ticketKey(ticket))}`;
}
