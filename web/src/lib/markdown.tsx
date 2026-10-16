import type { Components } from "react-markdown";

// Links in Markdown open in a new tab so the board stays open. Stopping the
// click keeps it from reaching a clickable parent, such as a project card.
export const markdownComponents: Components = {
  a: ({ href, title, children }) => (
    <a href={href} title={title} target="_blank" rel="noopener noreferrer" onClick={(e) => e.stopPropagation()}>
      {children}
    </a>
  ),
};
