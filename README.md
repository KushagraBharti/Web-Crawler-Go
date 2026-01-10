# Web Crawler in Go 

> **Status**: Early development. Learning Go and building this to understand systems engineering + networking performance.

## What I'm Building

A fast web crawler in Go with a live dashboard that shows you exactly what's happening, I want visualization that expose everything going on in the background. 

**The point**: Most crawlers are either libraries where behavior is opaque, or heavyweight stacks that are complex to understand. I want something more intuitive.

Also: perfect excuse to learn Go deeply.

## Why This Is Interesting

High-performance crawling can be hard. Here's what I think will be hard:
- Keeping concurrency high without being rude or unstable
- Handling redirects safely (no "politeness leaks")
- Making memory usage predictable (strict caps + backpressure)
- Getting TCP/TLS connection reuse right (huge for throughput)
- Avoiding thundering herd patterns (robots.txt is the classic)
- Building instrumentation that actually helps you tune

Go's `net/http` is great but only performs well if you reuse clients/transports and tune the defaults intentionally.

## Stack

- **Backend**: Go (`net/http`, bounded worker pools, streaming tokenizer)
- **Frontend**: Next.js + TypeScript + React
- **Storage**: Postgres (either Supabase or Convex... Or maybe I will challenge myself to just use Postgres directly)

## Why I'm Doing This & Step-by-Step Plan

Honestly, I'm building this as an excuse to get better at Go and systems programming. I needed a project that would force me to deal with concurrency and networking headaches so I can actually learn how to solve them.

### Usually, I'm trying to figure out...
- **How to manage resources**: How do I stop memory from exploding? How do I handle backpressure and scheduling fairly?
- **Go Networking**: I know `net/http` exists, but how do I reuse connections properly? How do I actually prove that connection reuse is happening?
- **Observability**: Instead of guessing why things are slow, I want metrics (latency, error rates) and a dashboard that visualizes what's happening.
- **Doing it "Right"**: Handling redirects without breaking politeness rules, respecting `robots.txt` without causing traffic jams, and avoiding crawler traps.

### My rough plan to build this

**Phase 0: The Setup (1 day)**  
Just getting the basics running: Docker Compose with Go, Postgres, and Next.js. I want a simple UI where I can just type in a URL and hit "Go".

**Phase 1: The MVP (5 days)**  
Building the smallest crawler that actually works. 
- A single pipeline: find link → fetch page → parse it → save it.
- Adding basic limits so I don't accidentally DDoS a site.
- Streaming HTML parsing (just grabbing `<a href>`).
- Hooking up everything so I can watch the crawl happen live on the dashboard.

**Phase 2: Making it reliable (2 weeks)**  
This is where I expect things to break, so I'll focus on fixing them.
- Handling redirects properly (re-queueing them instead of blindly following).
- Adding timeouts and strict limits on everything.
- Implementing retries with jitter and circuit breakers for broken hosts.
- Actually tracking connection reuse and exposing `pprof` to see where the bottlenecks are.

**Phase 3: Being a good citizen (3 days)**  
Adding `robots.txt` support so the crawler is polite. I need to make sure fetching robots files doesn't cause a "thundering herd" problem when I discover 500 new hosts at once.

**Phase 4: Polish (5 days)**  
Cleaning up the UI and maybe adding some cool stats, like identifying "host personalities" (e.g., is this site slow? Does it redirect a lot?).

This is a learning project—feedback on architecture, Go idioms, and performance tuning are welcome.