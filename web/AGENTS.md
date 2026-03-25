# AI Agent Guidelines

Welcome to the KVStore Dashboard project! If you are an AI assistant or agent working on this repository, please adhere strictly to the following rules to maintain code quality, avoid legacy patterns, and ensure a seamless developer experience.

## Project Architecture
- **Backend**: A custom distributed key-value store built in Go. Located in `<root>/server`. It exposes a REST API on `:8080` and a TCP protocol on `:6379`.
- **Frontend**: A Next.js 15 (App Router) dashboard located in `<root>/web`.
- **Communication**: Frontend communicates with the backend via REST (for CRUD and stats) and WebSockets (`/ws/events`) for real-time updates.

## Frontend Rules (Next.js 15 + Tailwind v4 + shadcn/ui)
1. **App Router Only**: We strictly use the Next.js App Router (`app/` directory). Do not use the deprecated Pages Router (`pages/`).
2. **Server vs. Client Components**: Default to Server Components. Use `"use client"` *only* when necessary (e.g., hooks, state, event listeners, Tanstack Query data fetching).
3. **Data Fetching**: We use `Tanstack Query` for all client-side data fetching, caching, and mutations (`hooks/useKeys.ts`, `hooks/useStats.ts`). Do not write bare `useEffect` fetches.
4. **Styling**: We use **Tailwind CSS v4**. Avoid legacy Tailwind config structures as v4 relies heavily on `@theme` and native CSS layers in `globals.css`.
5. **Component Library**: We use **shadcn/ui** with the Zinc / Dark theme. Always check if a shadcn component exists before building a custom one. Add them via the CLI `npx shadcn@latest add <component>`.
6. **Icons**: Use `lucide-react` for all icons.
7. **Design Tokens**: The design requirement is a "Dark terminal" aesthetic. Favor dark grays (`bg-zinc-950`), monospaced fonts for data, and high information density.
8. **Real-time Sync**: The dashboard subscribes to real-time WebSocket events. See `hooks/useEventStream.ts` for the implementation. Ensure any new data views handle these real-time signals.

## Backend Rules (Go)
1. **Concurrency**: The core engine uses mutexes (`sync.RWMutex`) to protect map data. Always ensure proper locking semantics.
2. **Persistence**: The server uses AOF (Append Only File) logging. Mutations must be logged to the AOF.
3. **Event Bus**: The server features an internal event bus. Mutations trigger events which are broadcasted to active WebSocket subscribers.

## AI Assistant Meta-Rules
1. **No Placeholders**: Do not write placeholder comments like `// implement logic here`. Provide fully functional code.
2. **Concise Explanations**: Provide the code along with a very brief explanation. Assume the user is an expert.
3. **Tool Selection**: Use the most specific tool for the job. Do not invent scripts when direct file edits will suffice.
4. **Hydration safety**: Be aware of browser extensions injecting classes into `<body>`. Use `suppressHydrationWarning` on the `html` and `body` tags in `app/layout.tsx`.
