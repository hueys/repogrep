# Chat Transcript: Local RAG App for Git Repo Search (Go/Python)

**Date:** August 14, 2026

---

**User:** I wanna create an app either in Go or Python that I can run locally on my MacBook Pro or MacBook Air. It does retrieval augmented generation. I want to ingest content from Git repos such as readme files and markdown and other metadata and then query the data to find projects that are related to certain topic or certain language. What are some good text stack options for Go and Python.

**Claude:** Good project. Let me think through this out loud for both stacks.

For Python, the classic combo would be using something like `sentence-transformers` for embeddings, a vector store like `Chroma` or `LanceDB` for local storage, and `LangChain` or just raw code for the retrieval and generation glue. Chroma's nice because it's dead simple to run locally with zero setup, and LanceDB is great if you want something fast and file-based without a server.

For Go, it's a bit thinner, but you'd typically pair something like a Go embedding client, calling out to Ollama locally for embeddings and generation, with a vector store like Qdrant, which has a nice Go client, or you could even use SQLite with a vector extension like `sqlite-vec` for something dead simple and file-based.

Given what you're describing — ingesting git repos, readmes, markdown, metadata — do you have a preference for keeping everything fully local and offline, or are you okay calling an API like OpenAI for the embeddings and generation part?

---

**User:** Would it be possible to use Claude?

**Claude:** Yeah, definitely. You'd use the Anthropic API for the generation step, feeding it the retrieved chunks as context, but you'd still need a separate embedding model since Claude itself doesn't do embeddings. So you'd pair Claude with something like Voyage AI's embeddings, which Anthropic actually recommends and has close ties with, or you could still use a local embedding model to keep costs down and only call Claude for the final generation step.

Want me to sketch out what that architecture would look like end to end?

---

**User:** What are good options for local embedding models?

**Claude:** Sure, a few solid ones. For general purpose text, BGE-base or BGE-large from BAAI are really popular right now and run great locally through sentence-transformers. Another great one is Nomic Embed, which is fast and specifically tuned to work nicely with Ollama if you want everything running through one local tool.

There's also all-MiniLM-L6-v2, which is smaller and faster but slightly less accurate — good if you want quick iteration during development. Given you're dealing with code and markdown rather than plain prose, do you want something general purpose, or should I look into embedding models specifically tuned for code?

---

**User:** As far as alternatives to RAG go, are there good options with databases and full text search?

**Claude:** Yeah, absolutely, and honestly for your use case that might even be a better starting point than full RAG. The simplest option is just SQLite with its built-in FTS5 extension, which gives you solid full text search with ranking, and it's a single file, zero setup — perfect for a local Mac app.

If you want something a bit more powerful, Postgres with its full text search, or the pg_trgm/ParadeDB extension, is great too, though that means running a database server rather than just a file. There's also Tantivy if you're leaning Rust-adjacent, or Bleve, which is a pure Go full text search library, so that could pair nicely if you go with the Go stack. Want me to compare full text search versus embeddings for your specific use case, like finding projects by topic or language?

---

**User:** Tell me more about Bleve for Go.

**Claude:** Ah, that's Bleve, B-L-E-V-E. It's a pure Go full text search and indexing library, kind of like an Apache Lucene equivalent but native to Go — no JVM or external server needed.

It handles indexing documents, ranking results with BM25 scoring, faceting, and even fuzzy and wildcard queries, all embedded directly in your app. For your use case, you'd index each repo's readme and metadata as a document, then query by keywords like a language name or topic, and it'd hand you back ranked results. Want me to sketch out what that indexing schema might look like for your git repo data?

---

**User:** Sure.

**Claude:** Okay, so picture each document in the Bleve index as one repo. You'd have fields like the repo name, the full readme text, a list of topics or tags, the primary language, and maybe the last updated date.

The readme text field would be the main one you search over with full text, while language and topics could be separate fields you use for filtering or boosting, so a query like "find machine learning projects in Python" would match the topic "machine learning" in the text, and then you'd filter or boost by the language field equaling Python. Want me to talk through how you'd actually pull that data from git repos to build these documents in the first place?

---

**User:** Yes, please.

**Claude:** Sure. So the ingestion step would walk through your local repos, or clone them if they're remote, and for each one pull out the readme file content, parse the package files like a package.json or a go.mod to figure out the primary language and dependencies, and grab metadata like the last commit date from the git log.

You'd then bundle all of that into one of those Bleve documents per repo and add it to the index, and you'd probably want a scheduled or on-demand refresh step so when repos update, you just re-index that one document instead of rebuilding everything from scratch.

---

**User:** Alright, thanks. Great time.

**Claude:** Sounds good! Have a great one.
