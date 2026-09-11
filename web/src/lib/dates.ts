// A due date is a calendar day, not an instant. The API serializes it as UTC
// midnight ("2026-09-28T00:00:00Z"), so new Date(value) would shift it to the
// previous day anywhere west of UTC. Build a local date from the YYYY-MM-DD part.
export const formatDueDate = (value: string, locale?: string) => {
  const [year, month, day] = value.slice(0, 10).split("-").map(Number);
  return new Date(year, month - 1, day).toLocaleDateString(locale);
};
