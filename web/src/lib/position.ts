// Board positions are sparse floats: a dropped card takes the midpoint of its
// new neighbours, or steps 1000 past the only one, so no other card moves.
export function positionBetween(before?: number, after?: number): number {
  if (before !== undefined && after !== undefined) return (before + after) / 2;
  if (before !== undefined) return before + 1000;
  if (after !== undefined) return after - 1000;
  return 0;
}
