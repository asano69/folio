## Why Folio Chooses Server-Side Rendering (SSR)

### What a URL Represents

In the Web architecture, a URL is not merely a navigation mechanism; it is a resource identifier. A URL such as `/books/{uuid}/pages/42` refers to a specific resource—“this page of this book.” That identifier should retain the same meaning today and in the future, regardless of browser changes or whether JavaScript is enabled.

With SSR, this principle holds naturally. When the server responds directly to `/books/{uuid}/pages/42`, that URL corresponds to a concrete endpoint registered in the Go router. Accessing the URL via `curl`, opening it from a bookmark, or linking to it from external notes all produce the same HTML response. The URL does not merely point to a resource; the URL is the resource.

This guarantee does not hold in a typical SPA architecture. The server returns the same `index.html` for all routes, and only the client-side JavaScript understands the meaning of the URL. If the frontend framework is replaced, the interpretation of those URLs becomes dependent on the new client-side router implementation. Links created earlier may become invalid. For example, a link stored in an external note may break in the future due to frontend changes. This contradicts Folio’s goal of guaranteeing persistent resource access.

### Cost and Return of Complexity

The choice of SSR is not based solely on URL design philosophy. When comparing the value provided by SPAs with Folio’s intended usage, there is no compelling justification for adopting SPA architecture.

SPAs provide the most value in scenarios requiring real-time updates, optimistic UI behavior, or offline capability. Folio is a self-hosted application typically used by a single user on their own server. Concurrent access is effectively limited to one user. There is no requirement for real-time updates. Offline capability provides no benefit, since if the server is unavailable, the materials themselves are also unavailable. None of the primary advantages of SPAs apply to this use case.

By contrast, adopting an SPA introduces unavoidable costs: framework selection and maintenance, client-side state management, duplicated type definitions across frontend and backend, and explicit API schema design for all endpoints. In the current design, Go templates embed data directly into HTML, eliminating this duplication. External dependencies are minimal, and no frontend framework is required.

From a development and operational perspective, the ability to run the entire system with a single `folio server` command is significant. Static file serving, HTML rendering, and API endpoints are handled within a single process. An SPA architecture would require running a development server alongside the Go server, increasing complexity and reducing the simplicity of the current build process, where JavaScript and CSS are built and the Go server is started.

### Consistent Design Philosophy

The use of CBZ, the choice of SQLite, minimizing TypeScript usage, and selecting SSR all follow the same principle: limit complexity to what is necessary and sufficient for the application’s purpose.

This principle aligns with the philosophy of URL permanence. Embedding UUIDs in `folio.json` within CBZ archives guarantees that a book’s identity persists even if the application disappears. Similarly, SSR guarantees that the same URL continues to return the same resource even if JavaScript implementations change.

The practical goal of minimizing complexity and the architectural goal of ensuring persistent resource access converge in the decision to use SSR.
