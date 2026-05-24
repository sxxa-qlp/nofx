# NOFX Simplification Plan

## Goal

Refactor NOFX from a broad "AI trading platform + agent platform + community product + multi-channel bot" into a **smaller, clearer, controllable trading system core** suitable for secondary development.

This document is **phase-0 architecture guidance** before code changes.

---

## Important Constraint

After simplification, the system **must still keep a frontend web UI** and the frontend **must continue to display the trading process**, not just static configuration.

That means the simplified version must still support:

1. trader runtime status display
2. account / equity display
3. position display
4. order display
5. AI decision / execution log display
6. trading process timeline / traceability from decision -> execution -> position/result

In other words:

> We are simplifying the system core, **not** turning it into a headless backend-only engine.

---

## First-Principles Definition

A minimal usable AI trading system only needs five essential capabilities:

1. **Account Access**
   - connect trading account
   - read balance, positions, orders

2. **Market Data**
   - fetch market prices / klines / indicator inputs

3. **Decision Engine**
   - transform market state + strategy into trading intent
   - AI or rule-based is acceptable

4. **Execution Engine**
   - convert intent into exchange actions
   - open / close / cancel / risk-control actions

5. **State Persistence**
   - persist users, traders, strategies, positions, orders, decisions, equity

Everything else is secondary.

---

## What NOFX Is Today

Today the repository combines multiple product layers:

- trading runtime
- strategy configuration
- AI decision logic
- multi-exchange adapters
- web API
- web dashboard
- Telegram bot
- general AI agent runtime
- MCP / provider / payment abstraction
- competition / public leaderboard
- onboarding / growth flows
- telemetry

This makes the system powerful, but also raises maintenance cost and architectural ambiguity.

---

## Current Core Structure

### 1. Bootstrap / Composition
- `main.go`

Responsibilities today:
- env loading
- config loading
- crypto init
- DB/store init
- TraderManager init
- API server startup
- NOFXi agent startup
- Telegram bot startup

Assessment:
- too many subsystems are started from one place
- composition root is overloaded

### 2. Persistence Layer
- `store/`

Key entities:
- user
- aiModel
- exchange
- trader
- decision
- position
- strategy
- equity
- order
- telegramConfig
- aiCharge
- grid

Assessment:
- this is a real core layer
- should be preserved, but narrowed to essential entities first

### 3. Runtime Orchestration
- `manager/`
- `trader/`

`manager/TraderManager`:
- loads traders from DB
- starts/stops trader instances
- restores running traders
- aggregates competition data

`trader/`:
- `auto_trader.go` and surrounding runtime files
- exchange-specific implementations
- paper trading implementation

Assessment:
- this is the heart of the product
- should remain the center of the simplified system

### 4. Market + Decision Preparation
- `market/`
- `kernel/`

`market/`:
- klines
- indicators
- market normalization

`kernel/`:
- prompt building
- decision schema
- analysis / decision organization

Assessment:
- should be preserved
- but boundaries should become cleaner and more explicit

### 5. API Layer
- `api/`

Current responsibilities include:
- auth
- user config
- trader config
- strategy config
- exchange config
- public leaderboard / competition
- Telegram config
- onboarding
- agent APIs
- wallet-related APIs

Assessment:
- too broad for a simplified core
- should be reduced to essential trading control APIs first

### 6. Frontend
- `web/`

Assessment:
- must be preserved
- but should focus on trading control and trading process visibility
- should not remain responsible for every growth/community surface in phase 1

### 7. Secondary Platforms
- `agent/`
- `telegram/`
- `mcp/`
- `telemetry/`

Assessment:
- valuable, but not required for the minimal trading-system core
- should be frozen or detached first

---

## Simplification Principle

We are **not** deleting features just because they exist.
We are deleting/freezing anything that is not required for the minimal closed loop:

> market input -> strategy/AI decision -> execution -> persistence -> frontend-visible trading process

---

## Modules to Preserve

## A. Must Keep

### `store/`
Keep these entities first:
- user
- exchange
- trader
- strategy
- decision
- order
- position
- equity
- system_config

### `manager/`
Keep:
- trader lifecycle management
- trader loading / start / stop / restore

But simplify away non-core aggregation responsibilities later.

### `trader/`
Keep:
- `auto_trader*`
- `types`
- `paper`
- selected real exchange adapters

### `market/`
Keep:
- market data fetching
- kline processing
- indicators needed by runtime decision flow

### `kernel/`
Keep:
- decision schema
- prompt/decision construction
- validation

### `api/`
Keep only core control-plane APIs in the simplified phase.

### `web/`
Keep web UI with clear support for trading-process visibility.

---

## B. Frontend Capabilities That Must Survive Simplification

The simplified frontend must still show the **trading process**, not only trader config.

### Required UI surfaces

1. **Trader Overview**
   - trader list
   - running/stopped status
   - selected strategy / exchange / model

2. **Account & Equity**
   - current balance
   - equity curve
   - unrealized pnl
   - return percentage

3. **Positions**
   - open positions
   - side / size / entry / mark / pnl / leverage

4. **Orders**
   - open orders
   - historical orders
   - fills

5. **Decision Log**
   - AI decision records
   - symbol / action / confidence / reasoning / time

6. **Execution Trace / Trading Process Timeline**
   - decision created
   - execution attempted
   - order placed / rejected
   - position opened / closed
   - result reflected in equity/position state

### Product constraint

If we remove some modules, we must **not** remove the data flow required to support the above pages.

---

## Modules to Freeze or Remove First

## A. Freeze First (Phase 1)

### `telegram/`
Reason:
- channel integration, not trading core
- increases async complexity and support surface

Action:
- stop booting it in `main.go`
- stop exposing telegram config UI in first simplified phase
- keep source code in repo initially, but do not activate it

### `agent/`
Reason:
- broad general-purpose agent runtime
- much larger than needed for a first clean trading core
- obscures core decision/runtime boundaries

Action:
- stop booting it in `main.go`
- stop exposing `/api/agent/*` in phase 1
- replace with a thinner AI decision abstraction later if needed

### `telemetry/`
Reason:
- not required for minimal closed loop

Action:
- disable installation telemetry path during simplification phase

### public competition / leaderboard paths
Reason:
- showcase/community feature
- not required for minimal product loop

Action:
- freeze public competition routes and related frontend entry points in phase 1

### onboarding / beginner mode
Reason:
- growth funnel, not runtime core

Action:
- freeze onboarding-specific flows in phase 1

---

## B. Simplify Heavily Later

### `mcp/`
Reason:
- current provider/payment abstraction is broader than necessary for phase-1 architecture cleanup

Action:
- keep runtime-compatible code if needed for existing AI calls
- but conceptually reduce future design to a smaller `DecisionProvider` abstraction
- do not let payment/provider concerns dominate runtime design

---

## Exchange Scope Reduction

Current repository supports many exchanges.
That is useful commercially, but too broad for a first simplification pass.

## Recommended phase-1 exchange scope

Keep active support for:
- `paper`
- one primary real CEX: `binance` **or** `okx`
- optionally one DEX if strategically required: `hyperliquid`

Freeze/hide the rest from phase-1 UI and API exposure.

This reduces:
- testing matrix size
- exchange-specific bug surface
- config complexity
- runtime branching

---

## Minimal Target Architecture

A cleaner future structure should converge toward:

### `core/store`
- persistence models
- repositories
- database integration

### `core/market`
- market fetchers
- indicator generation
- normalized market snapshots

### `core/decision`
- strategy interpretation
- prompt/schema generation
- AI/rule decision interface

### `core/execution`
- exchange abstraction
- order/position synchronization
- execution + risk controls

### `core/runtime`
- trader lifecycle
- scan loop
- scheduling
- runtime state machine

### `app/api`
- trading control plane

### `app/web`
- trader configuration + trading process visibility

This is a direction, not an immediate directory rename mandate.

---

## Phase Plan

## Phase 0 — Documentation and boundary agreement

Deliverables:
- this plan
- explicit preservation of frontend trading-process visibility
- explicit list of frozen modules

## Phase 1 — Non-destructive freezing

Do not delete code yet.

### Changes
1. stop starting `nofxiAgent` in `main.go`
2. stop starting `telegram` in `main.go`
3. disable agent routes from API boot path
4. disable telegram routes from API boot path
5. disable competition/public leaderboard routes from phase-1 surface
6. disable onboarding routes from phase-1 surface
7. hide corresponding frontend entry points

### Result
A smaller product:
- web UI
- API server
- trader runtime
- persistence
- market data
- AI decision path

## Phase 2 — Exchange narrowing

### Changes
1. keep `paper` always
2. expose only selected real exchanges in phase-1 UI/API
3. mark others as experimental/hidden

### Result
A smaller, testable exchange matrix.

## Phase 3 — AI layer thinning

### Changes
1. define a smaller decision-provider abstraction
2. isolate market input -> decision output interface
3. reduce dependence on broad general-purpose agent runtime

### Result
Cleaner runtime reasoning flow.

## Phase 4 — Persistence cleanup

### Keep
- users
- traders
- strategies
- exchanges
- positions
- orders
- decisions
- equity
- system_config

### Freeze / review
- telegram_config
- ai_charge
- visibility/competition-specific state
- onboarding-specific state

---

## API Scope for Simplified Phase

## Keep
- auth
- exchange CRUD
- model CRUD (if AI remains configurable via UI)
- strategy CRUD
- trader CRUD
- trader start/stop
- account
- positions
- position history
- orders
- fills
- decisions
- statistics
- equity history

## Freeze / hide
- agent routes
- telegram routes
- competition/public leaderboard
- onboarding/beginner mode
- wallet helper flows not needed for trading runtime

---

## Frontend Scope for Simplified Phase

## Keep
- login/auth shell
- trader list
- trader config/edit/create
- strategy config/edit/create
- exchange config/edit/create
- dashboard / account
- positions
- orders / fills
- decision logs
- equity chart
- trading process / traceability views

## Freeze / hide
- onboarding wizard
- telegram UI
- agent chat UI
- community competition surfaces
- extra growth/product pages not needed for runtime control

---

## Risks

### 1. Hidden coupling between runtime and frozen modules
Example:
- some API or frontend pages may indirectly rely on competition or telegram state

Mitigation:
- freeze by disabling boot path and routes first, not by hard delete

### 2. UI still needs enough data for trading process display
If we simplify too aggressively, frontend becomes config-only.

Mitigation:
- explicitly preserve decision/order/position/equity data paths
- verify dashboard pages before removing anything

### 3. AI provider path may currently depend on broader MCP abstractions
Mitigation:
- do not rewrite AI provider stack in phase 1
- first isolate, then reduce

### 4. TraderManager currently mixes runtime orchestration and competition aggregation
Mitigation:
- in later phase, separate runtime orchestration concerns from public ranking concerns

---

## Success Criteria

The simplification is successful when:

1. backend starts without agent/telegram/competition/onboarding subsystems
2. web UI still works as the main operator interface
3. traders can still be created, started, stopped, and observed
4. frontend can still display the trading process end-to-end
5. paper trading still works
6. at least one real exchange path still works
7. code ownership and module boundaries are clearer than before

---

## Recommended Immediate Next Step

Proceed with **Phase 1 non-destructive freezing**.

Concrete next implementation batch:

1. checkpoint current git state
2. stop `agent` boot in `main.go`
3. stop `telegram` boot in `main.go`
4. disable related API route registration
5. hide related frontend entry points
6. verify build
7. verify dashboard still shows:
   - account
   - positions
   - orders
   - decisions
   - equity/trading process

---

## Final Position

We are not trying to make NOFX smaller for its own sake.
We are trying to make it:

- clearer
- more maintainable
- easier to reason about
- easier to extend safely
- still visibly useful through the frontend

So the right target is:

> **A smaller core system with a retained trading dashboard and retained trading-process visibility.**
