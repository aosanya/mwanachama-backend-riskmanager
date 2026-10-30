# Importable libraries

A library is a data file, not a spec: a bow-tie's worth of risk, pathways and controls that a mount can load into a register. `library/ai_safety.library.json` is the first, holding the owner's enterprise AI safety conversation (see [sources.md](../1.%20requirements/sources.md)): the top event, its eight threat pathways with their example scenarios, the barrier and detection-and-containment table as controls, and the drill-down of the retrieval pathway into a sub-bow-tie of six threats.

What it does not hold, and says so in its `note`: any consequence, because the conversation drew them only in an image; any control on the logging and cross-tenant pathways, because the conversation's table had no rows for them; and any control on the retrieval sub-bow-tie, because the conversation named none. Each control is placed in the order the conversation listed it, preventive first, then detection and containment, and carries its stage in the control's document.

`LoadLibrary(path)` reads one, `Library.Check(spec)` runs every risk, pathway and control row through `Check` against the domain spec with no connection, and `ImportLibrary(ctx, spec, library, open)` calls `Check` first and only then calls `open` for a manager. A row that fails is an `ErrInvalid` and nothing is opened or written.

The import is idempotent. A risk is matched on its title and the pathway it drills into, a pathway on its title within its risk, a control on its name, a barrier on its pathway and control. What is found is counted as existing and left as it is, so a second run creates nothing and a register edited since the first run keeps its edits. The risk is created as a draft with no owner; naming one is the mount's job, and until then it shows as a gap.

Tests: `library_test.go`.
