import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useSensor,
  useSensors,
  useDroppable,
  useDraggable,
  closestCorners,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
  type UniqueIdentifier,
} from "@dnd-kit/core";
import {
  Calendar,
  AlertTriangle,
  ArrowUp,
  ArrowRight,
  ArrowDown,
  CheckCircle2,
  FolderKanban,
  Users,
  Plus,
} from "lucide-react";
import { api, type Ticket, type TicketInput, type Project, type Team, type BoardColumn, type Label } from "../api/client";
import TicketPanel from "../components/TicketPanel";
import CreateTicketModal from "../components/CreateTicketModal";
import LabelChip from "../components/LabelChip";
import { STATUSES, STATUS_LABELS } from "../constants/statuses";

const PRIORITY_CONFIG: Record<string, { color: string; icon: typeof ArrowUp }> = {
  urgent: { color: "text-red-600", icon: AlertTriangle },
  high: { color: "text-orange-600", icon: ArrowUp },
  medium: { color: "text-amber-600", icon: ArrowRight },
  low: { color: "text-green-600", icon: ArrowDown },
};

function PriorityBadge({ priority }: { priority: string }) {
  const config = PRIORITY_CONFIG[priority];
  if (!config) return null;
  const Icon = config.icon;
  return (
    <span
      role="img"
      aria-label={`${priority} priority`}
      title={`${priority} priority`}
      className={`inline-flex shrink-0 items-center ${config.color}`}
    >
      <Icon aria-hidden="true" className="h-3.5 w-3.5" />
    </span>
  );
}

function SubtaskProgress({ subtasks }: { subtasks: Ticket["subtasks"] }) {
  if (!subtasks || subtasks.length === 0) return null;
  const done = subtasks.filter((candidate) => candidate.completed).length;
  const pct = Math.round((done / subtasks.length) * 100);
  return (
    <div
      title={`${done} of ${subtasks.length} subtasks complete`}
      className="flex shrink-0 items-center gap-1.5 text-[11px] text-slate-500"
    >
      <CheckCircle2 aria-hidden="true" className="h-3 w-3" />
      <div className="h-1 w-8 overflow-hidden rounded-full bg-slate-200">
        <div
          className="h-full rounded-full bg-blue-500 transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span>
        {done}/{subtasks.length}
      </span>
    </div>
  );
}

function TicketCard({
  ticket,
  projects,
  teams,
  isDragging,
  onClick,
}: {
  ticket: Ticket;
  projects: Project[];
  teams: Team[];
  isDragging?: boolean;
  onClick?: () => void;
}) {
  const project = projects.find((candidate) => candidate.id === ticket.projectId);
  const team = teams.find((candidate) => candidate.id === ticket.teamId);

  return (
    <div
      onClick={onClick}
      className={`cursor-pointer space-y-2 rounded-[5px] border bg-white px-3 py-2.5 transition-[border-color,box-shadow] ${
        isDragging
          ? "border-blue-400 shadow-[0_8px_20px_rgba(9,30,66,0.24)]"
          : "border-slate-300 shadow-[0_1px_1px_rgba(9,30,66,0.12)] hover:border-blue-400 hover:shadow-[0_2px_4px_rgba(9,30,66,0.16)]"
      }`}
    >
      <p className="text-[13px] font-medium leading-[1.35] text-slate-800">
        {ticket.title}
      </p>
      {ticket.labels && ticket.labels.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {ticket.labels.map((label) => (
            <LabelChip key={label.id} label={label} variant="board" />
          ))}
        </div>
      )}
      {(project || team) && (
        <div className="flex flex-wrap items-center gap-1.5">
          {project && (
            <span
              className="inline-flex items-center gap-1 rounded-[3px] px-1.5 py-0.5 text-[11px] font-medium text-slate-700"
              style={{
                backgroundColor: (project.color || "#3b82f6") + "14",
              }}
            >
              <FolderKanban
                aria-hidden="true"
                className="h-3 w-3"
                style={{ color: project.color || "#3b82f6" }}
              />
              {project.name}
            </span>
          )}
          {team && (
            <span
              className="inline-flex items-center gap-1 rounded-[3px] px-1.5 py-0.5 text-[11px] font-medium text-slate-700"
              style={{
                backgroundColor: (team.color || "#8b5cf6") + "14",
              }}
            >
              <Users
                aria-hidden="true"
                className="h-3 w-3"
                style={{ color: team.color || "#8b5cf6" }}
              />
              {team.name}
            </span>
          )}
        </div>
      )}
      <div className="flex min-w-0 items-center justify-between gap-2">
        <span className="shrink-0 text-[11px] font-medium text-slate-500">
          {ticket.projectPrefix}-{ticket.number}
        </span>
        <div className="flex min-w-0 flex-wrap items-center justify-end gap-2">
          <PriorityBadge priority={ticket.priority} />
          {ticket.dueDate && (
            <span
              title="Due date"
              className="inline-flex shrink-0 items-center gap-1 text-[11px] text-slate-500"
            >
              <Calendar aria-hidden="true" className="h-3 w-3" />
              {new Date(ticket.dueDate).toLocaleDateString()}
            </span>
          )}
          <SubtaskProgress subtasks={ticket.subtasks} />
        </div>
      </div>
    </div>
  );
}

function DraggableTicket({
  ticket,
  projects,
  teams,
  onClick,
}: {
  ticket: Ticket;
  projects: Project[];
  teams: Team[];
  onClick: () => void;
}) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: ticket.id,
    data: { ticket },
  });

  return (
    <div
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      className={`cursor-grab active:cursor-grabbing ${isDragging ? "opacity-30" : ""}`}
    >
      <TicketCard ticket={ticket} projects={projects} teams={teams} onClick={onClick} />
    </div>
  );
}

function Column({
  status,
  tickets,
  projects,
  teams,
  onTicketClick,
  onAddTicket,
  totalCount = tickets.length,
  filtered = false,
}: {
  status: string;
  tickets: Ticket[];
  projects: Project[];
  teams: Team[];
  onTicketClick: (ticket: Ticket) => void;
  onAddTicket: (status: string) => void;
  totalCount?: number;
  filtered?: boolean;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: status });
  const statusLabel = STATUS_LABELS[status];
  const countLabel = filtered ? `${tickets.length}/${totalCount}` : `${totalCount}`;

  return (
    <section
      aria-label={`${statusLabel}, ${countLabel} tickets`}
      className="flex min-w-56 flex-1 flex-col overflow-hidden rounded-md bg-slate-100"
    >
      <div className="flex items-center gap-1.5 px-3 py-2.5">
        <h3 className="text-[11px] font-semibold uppercase tracking-wide text-slate-600">
          {statusLabel}
        </h3>
        <span className="rounded-full bg-slate-200 px-1.5 py-0.5 text-[10px] font-medium leading-none text-slate-600">
          {countLabel}
        </span>
        <button
          type="button"
          aria-label={`Add ticket to ${statusLabel}`}
          onClick={() => onAddTicket(status)}
          className="ml-auto rounded p-1 text-slate-500 transition-colors hover:bg-slate-200 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
        >
          <Plus aria-hidden="true" className="h-4 w-4" />
        </button>
      </div>
      <div
        ref={setNodeRef}
        className={`min-h-0 flex-1 space-y-1.5 overflow-y-auto px-2 pb-2 transition-shadow ${
          isOver ? "ring-2 ring-inset ring-blue-500" : ""
        }`}
      >
        {tickets.map((ticket) => (
          <DraggableTicket
            key={ticket.id}
            ticket={ticket}
            projects={projects}
            teams={teams}
            onClick={() => onTicketClick(ticket)}
          />
        ))}
        {tickets.length === 0 && (
          <div className="flex h-24 items-center justify-center rounded-[5px] border border-dashed border-slate-300 text-xs text-slate-500">
            Drop tickets here
          </div>
        )}
      </div>
    </section>
  );
}

export default function Board() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [selectedProject, setSelectedProject] = useState<string>("");
  const [columns, setColumns] = useState<BoardColumn[]>([]);
  const [activeTicket, setActiveTicket] = useState<Ticket | null>(null);
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [createForStatus, setCreateForStatus] = useState<string | null>(null);
  const [filterLabelId, setFilterLabelId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const loadRequestGeneration = useRef(0);

  const labelsInPlay = useMemo(() => {
    const labelsById = new Map<string, Label>();

    for (const column of columns) {
      for (const ticket of column.tickets) {
        for (const label of ticket.labels || []) {
          labelsById.set(label.id, label);
        }
      }
    }

    return Array.from(labelsById.values()).sort((a, b) =>
      a.name.localeCompare(b.name)
    );
  }, [columns]);

  const effectiveFilterLabelId = labelsInPlay.some(
    (label) => label.id === filterLabelId
  )
    ? filterLabelId
    : null;

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } })
  );

  const loadBoard = useCallback(async () => {
    const requestGeneration = ++loadRequestGeneration.current;

    try {
      const board = await api.board.get(selectedProject || undefined);
      if (requestGeneration !== loadRequestGeneration.current) return;

      const nextColumns = board.columns || [];
      setColumns(nextColumns);
      setFilterLabelId((current) =>
        current &&
        nextColumns.some((column) =>
          column.tickets.some((ticket) =>
            ticket.labels?.some((label) => label.id === current)
          )
        )
          ? current
          : null
      );
    } catch {
      if (requestGeneration !== loadRequestGeneration.current) return;

      setColumns(
        STATUSES.map((status) => ({ status, tickets: [] }))
      );
      setFilterLabelId(null);
    }
    setLoading(false);
  }, [selectedProject]);

  useEffect(() => {
    api.projects.list().then(setProjects).catch(() => setProjects([]));
    api.teams.list().then(setTeams).catch(() => setTeams([]));
  }, []);

  useEffect(() => {
    setLoading(true);
    loadBoard();
  }, [loadBoard]);

  const getColumnTickets = (status: string) => {
    const tickets = columns.find((c) => c.status === status)?.tickets || [];
    if (!effectiveFilterLabelId) return tickets;
    return tickets.filter((ticket) =>
      ticket.labels?.some((label) => label.id === effectiveFilterLabelId)
    );
  };

  const getColumnTotal = (status: string) =>
    columns.find((column) => column.status === status)?.tickets.length ?? 0;

  const findTicketById = (id: UniqueIdentifier): Ticket | undefined => {
    for (const col of columns) {
      const found = col.tickets.find((t) => t.id === id);
      if (found) return found;
    }
    return undefined;
  };

  const findColumnByTicketId = (id: UniqueIdentifier): string | undefined => {
    for (const col of columns) {
      if (col.tickets.find((t) => t.id === id)) return col.status;
    }
    return undefined;
  };

  const handleDragStart = (event: DragStartEvent) => {
    const ticket = findTicketById(event.active.id);
    setActiveTicket(ticket ?? null);
  };

  const clearActiveDrag = () => setActiveTicket(null);

  const handleDragCancel = () => {
    clearActiveDrag();
    loadBoard();
  };

  const handleDragOver = (event: DragOverEvent) => {
    const { active, over } = event;
    if (!over) return;

    const activeStatus = findColumnByTicketId(active.id);
    const overStatus = STATUSES.includes(over.id as string)
      ? (over.id as string)
      : findColumnByTicketId(over.id);

    if (!activeStatus || !overStatus || activeStatus === overStatus) return;

    setColumns((prev) =>
      prev.map((col) => {
        if (col.status === activeStatus) {
          return { ...col, tickets: col.tickets.filter((t) => t.id !== active.id) };
        }
        if (col.status === overStatus) {
          const ticket = findTicketById(active.id);
          if (!ticket) return col;
          return { ...col, tickets: [...col.tickets, { ...ticket, status: overStatus }] };
        }
        return col;
      })
    );
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    clearActiveDrag();
    const { active, over } = event;

    if (!over) {
      loadBoard();
      return;
    }

    const targetStatus = STATUSES.includes(over.id as string)
      ? (over.id as string)
      : findColumnByTicketId(over.id);

    if (!targetStatus) {
      loadBoard();
      return;
    }

    try {
      await api.tickets.move(active.id as string, targetStatus);
    } catch {
      loadBoard();
    }
  };

  const handleTicketClick = (ticket: Ticket) => {
    setSelectedTicket(ticket);
  };

  const handleUpdate = async (id: string, data: TicketInput) => {
    await api.tickets.update(id, data);
    loadBoard();
  };

  const handleDelete = async (id: string) => {
    await api.tickets.delete(id);
    loadBoard();
  };

  const handleCreate = async (data: TicketInput) => {
    await api.tickets.create(data);
    setCreateForStatus(null);
    loadBoard();
  };

  return (
    <div className="flex h-full min-h-0 flex-col bg-slate-50 text-slate-900">
      <header className="flex h-14 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-5">
        <h1 className="text-lg font-semibold text-slate-900">Board</h1>
        <select
          value={selectedProject}
          onChange={(e) => setSelectedProject(e.target.value)}
          className="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-700 shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">All Projects</option>
          {projects.map((p) => (
            <option key={p.id} value={p.id}>
              {p.icon} {p.name}
            </option>
          ))}
        </select>
      </header>

      {labelsInPlay.length > 0 && (
        <div
          role="group"
          aria-labelledby="board-label-filter-label"
          className="flex shrink-0 flex-wrap items-center gap-2 border-b border-slate-200 bg-white px-5 py-2"
        >
          <span id="board-label-filter-label" className="text-xs text-slate-600">
            Filter by label:
          </span>
          {labelsInPlay.map((label) => (
            <LabelChip
              key={label.id}
              label={label}
              variant="board"
              active={effectiveFilterLabelId === label.id}
              onClick={() =>
                setFilterLabelId((current) =>
                  current === label.id ? null : label.id
                )
              }
            />
          ))}
          {effectiveFilterLabelId && (
            <button
              type="button"
              aria-label="Clear label filter"
              onClick={() => setFilterLabelId(null)}
              className="rounded px-1.5 py-0.5 text-xs text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
            >
              Clear
            </button>
          )}
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-x-auto overflow-y-hidden bg-slate-50 p-3">
        {loading ? (
          <div className="flex h-full items-center justify-center text-slate-500">
            Loading board…
          </div>
        ) : (
          <DndContext
            sensors={sensors}
            collisionDetection={closestCorners}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDragEnd={handleDragEnd}
            onDragCancel={handleDragCancel}
          >
            <div className="flex h-full min-w-[72rem] gap-2">
              {STATUSES.map((status) => (
                <Column
                  key={status}
                  status={status}
                  tickets={getColumnTickets(status)}
                  totalCount={getColumnTotal(status)}
                  filtered={effectiveFilterLabelId !== null}
                  projects={projects}
                  teams={teams}
                  onTicketClick={handleTicketClick}
                  onAddTicket={setCreateForStatus}
                />
              ))}
            </div>
            <DragOverlay>
              {activeTicket ? (
                <TicketCard ticket={activeTicket} projects={projects} teams={teams} isDragging />
              ) : null}
            </DragOverlay>
           </DndContext>
        )}
      </div>

      {createForStatus && (
        <CreateTicketModal
          projects={projects}
          teams={teams}
          defaultStatus={createForStatus}
          onClose={() => setCreateForStatus(null)}
          onCreate={handleCreate}
        />
      )}

      {selectedTicket && (
        <TicketPanel
          ticket={selectedTicket}
          projects={projects}
          teams={teams}
          onClose={() => {
            setSelectedTicket(null);
            loadBoard();
          }}
          onUpdate={handleUpdate}
          onDelete={handleDelete}
        />
      )}
    </div>
  );
}
