# Claude / AI Assistant Rules

## Tech Stack Overview
- **Next.js 15** (App Router)
- **React 19**
- **TypeScript**
- **Tailwind CSS v4**
- **shadcn/ui**
- **Tanstack Query** (React Query)
- **Go** (Custom KV Store Backend)

## Coding Standards

### React & Next.js
- Use Server Components by default.
- Add `"use client"` at the very top of files only when using React hooks, event listeners, or Tanstack Query hooks.
- Prefer Functional Components with standard arrow functions or `function` declarations.
- Keep components small, focused, and purely functional. Extract heavy logic into custom hooks.
- Use `lucide-react` for iconography.

### State Management & Fetching
- **Never** use naked `fetch` inside a `useEffect` for data.
- **Always** use Tanstack Query (`useQuery`, `useMutation`) for syncing server state.
- Keep API call implementations cleanly separated in `lib/api.ts`.
- Types matching the Go server structs MUST reside in `lib/types.ts`.

### Styling
- The mandatory aesthetic is a **minimalist dark terminal vibe**.
- Rely on `zinc` colors exclusively (`bg-zinc-950`, `text-zinc-400`, etc).
- Use Tailwind CSS v4 features. Do not try to modify standard `tailwind.config.js` patterns, since v4 builds config directly into `globals.css` using `@theme`.
- Use the `cn()` utility (`lib/utils.ts`) for conditionally merging Tailwind classes.

### Backend (Go)
- The server runs an HTTP API on `:8080`, exposing REST endpoints (`/api/keys`) and a WebSocket stream (`/ws/events`).
- The WebSocket stream pushes JSON envelopes with type `SET`, `DEL`, `EXPIRE`, `EXPIRED`. Handle these strictly typing them in the frontend.

## General AI Instructions
- Stop and think before outputting code.
- If you find a bug, fix it immediately without waiting for permission.
- Write absolute paths if referencing files in outputs.
- Maintain the exact, dense, and functional design language of the dashboard. Do not add unnecessary fluff, colors, or padding that breaks the terminal aesthetic.
- Never output truncated code (e.g., `...existing code...`). Output the complete, drop-in replacement where possible, or use explicit surgical edits.
