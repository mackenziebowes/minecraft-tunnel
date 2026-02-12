# Stage-Actor-Prop Metaphor for Code Documentation

The Stage-Actor-Prop metaphor provides a cognitive framework for explaining code in a way that resonates with both human and LLM understanding. It mirrors how both humans and models process contextual information.

## Core Concepts

### Stage
The **Stage** represents the execution context or environment where code operates.

**Examples:**
- A function call stack (Stage: `processData()` function execution)
- A React component render (Stage: `UserProfile` component mounting)
- A database transaction (Stage: SQL query execution)
- An event handler (Stage: click event propagation)

**When describing the Stage:**
- What context is active?
- What is the entry point?
- What environment variables or state are present?
- What is the temporal context (before, during, after)?

### Actor
The **Actor** represents the active entity—the function, class, method, or module performing work.

**Examples:**
- `validateUser()` function (Actor: validation logic)
- `UserService` class (Actor: user management abstraction)
- `formatDate()` utility (Actor: date formatting logic)
- `useAuth()` hook (Actor: authentication state manager)

**When describing the Actor:**
- What is its primary responsibility?
- What transformation does it perform?
- What inputs does it accept?
- What outputs does it produce?

### Prop
The **Prop** represents the data structures, utilities, or dependencies the Actor uses.

**Examples:**
- `user: User` object (Prop: user data)
- `axios` HTTP client (Prop: network utility)
- `formatCurrency()` helper (Prop: formatting utility)
- `config` object (Prop: configuration data)

**When describing Props:**
- What is the structure of the data?
- What operations does the utility provide?
- Why is this dependency necessary?
- What constraints or assumptions does it have?

## Sentence Template

Use this sentence structure consistently:

> "In this **[Stage]** context, **[Actor]** uses **[Prop]** to **[action/intention]**."

## Examples

### Simple Function

```typescript
function validateEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}
```

> "In this **function call** context, **`validateEmail()`** uses **the regex pattern** to **determine if the input string is a valid email address.**"

### React Component

```tsx
function UserCard({ user }: { user: User }) {
  return (
    <div className="card">
      <h2>{user.name}</h2>
      <p>{user.email}</p>
    </div>
  );
}
```

> "In this **component render** context, **`UserCard`** uses **the `user` prop** to **display user information in a card layout.**"

### Class Method

```python
class DataProcessor:
    def __init__(self, client: HttpClient):
        self.client = client

    def fetch_data(self, url: str) -> dict:
        response = self.client.get(url)
        return response.json()
```

> "In this **method execution** context, **`fetch_data()`** uses **the HTTP client** to **retrieve and parse data from a remote URL.**"

### Complex Context

```go
func (s *Service) ProcessTransaction(ctx context.Context, tx Transaction) error {
    if err := s.validator.Validate(tx); err != nil {
        return err
    }
    return s.repo.Save(ctx, tx)
}
```

> "In this **service method** context, **`ProcessTransaction()`** uses **the validator** to **ensure transaction validity**, then uses **the repository** to **persist the transaction.**"

## When to Use This Metaphor

Use Stage-Actor-Prop when documenting:

1. **Function logic** - What a function does and how
2. **Component behavior** - How React/Vue/components render data
3. **Service methods** - How business logic is applied
4. **Data flows** - How data moves through the system
5. **Integrations** - How external dependencies are used

## Benefits

- **Human understanding**: Mirrors natural storytelling structure
- **LLM alignment**: Matches transformer attention patterns (context → entity → action)
- **Consistency**: Provides uniform documentation style
- **Clarity**: Separates concerns (context, actor, tools, outcome)
- **Intention focus**: Emphasizes purpose over implementation details

## Common Patterns

### Multiple Props

> "In this **API handler** context, **`createOrder()`** uses **the request body**, **the validator**, and **the database client** to **create a validated order record.**"

### Nested Actors

> "In this **render phase** context, **`DataTable`** uses **the `Column` components** to **display tabular data,** where each **`Column`** uses **the `data` prop** to **render individual cells.**"

### Conditional Logic

> "In this **validation flow** context, **`checkPermissions()`** uses **the user's role** to **determine access rights,** then uses **the permission matrix** to **grant or deny the requested action.**"

## Anti-Patterns to Avoid

- ❌ Describing implementation details instead of intention ("loops through array" → "processes collection")
- ❌ Mixing stage and prop ("in the database" is a stage, "the user object" is a prop)
- ❌ Using passive voice ("data is validated by" → "validator validates")
- ❌ Omitting the intention ("uses X" → "uses X to accomplish Y")

## Integration with Documentation Files

When writing sidecar markdown files, use Stage-Actor-Prop sentences to explain:

1. **Purpose** - Why this code exists
2. **Behavior** - What the code does
3. **Interactions** - How code relates to dependencies
4. **Data flow** - How data enters, transforms, and exits

Each major function, class, or component should have at least one Stage-Actor-Prop sentence describing its primary behavior.
