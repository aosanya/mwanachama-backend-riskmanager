# Sources

Every source the design used. Add to this page when a new one is used.

## The owner's AI safety bow-tie conversation

**URL:** https://chatgpt.com/share/6adf3e7a-2560-83eb-bf01-5c41839ed633
**Title of the shared page:** "Add Enterprise AI Safety Bow Tie"
**Read:** 2026-10-26.

The shared page is rendered by the browser, so a plain fetch returns only
its title. The conversation text was recovered on 2026-10-26 by decoding the
data embedded in the page's HTML. What follows is what that text says, and
nothing more.

**What the page carries as text.** Three exchanges. In the first, the owner
supplied a picture of an existing bow-tie and asked for it to be added to
"for enterprise AI safety"; the reply was an image. In the second, the owner
asked to "drill down and put data leaks in the middle for a start"; the
reply was prose and a second image. In the third, the owner said "Ok dig in
deeper"; the reply was a long structured answer and a third image. The three
images carry no text in the page data, so what they drew is not recorded
here.

**Exchange two, in the prose:**
- Take *Data Leak* out of the threats column and make it the centre as the
  specific risk event. The bow-tie then answers three questions: on the left,
  causes ("How could sensitive enterprise data be exposed through AI?"); in
  the middle, the top event ("Loss of control over confidential or sensitive
  data through an AI system"); on the right, consequences ("What happens if
  the data is exposed?").
- Distinguish **preventive controls** (stopping the leak) from **mitigative
  controls** (limiting the damage after a leak).
- Refine the causes to include employees pasting confidential data into
  public AI tools, sensitive information being retrieved through retrieval
  augmented generation (RAG), prompt injection causing unauthorised
  disclosure, and AI agents sending information to external systems.
- Data loss prevention and access restrictions generally sit on the
  preventive side when they stop disclosure. Once a leak has occurred,
  mitigative controls focus on revoking access, containing exposure,
  notifying affected parties and recovering.
- The result should support assessing "which controls exist, how effective
  they are, and where the gaps are".

**Exchange three, in the prose:**
- **Top event**, stated precisely: "Loss of control over sensitive enterprise
  data through AI" — sensitive information is disclosed, made accessible or
  transmitted to an unauthorised party or system through an AI application,
  model, agent or connected service.
- **Eight threat pathways**, each with an example failure scenario: prompt
  input leakage; RAG / retrieval leakage; prompt injection; agent / tool
  misuse; model output leakage; provider and integration leakage; logging
  and observability leakage; cross-tenant / session leakage. Their relevance
  depends on whether the enterprise uses public chatbots, internal copilots,
  RAG systems or autonomous agents.
- **A barrier table** with, per pathway, preventive barriers and "detection
  and containment". It covers six of the eight pathways — prompt input, RAG,
  prompt injection, agents, model output and third parties — and has no
  rows for logging and observability, or cross-tenant and session leakage.
  Example barriers: approved AI gateways, data classification, input DLP;
  document-level authorisation, permission-aware retrieval, tenant
  isolation; treating retrieved content as untrusted, tool authorisation,
  restricted network egress; least-privilege tools, approval gates,
  destination allowlists, scoped credentials; output DLP; vendor due
  diligence, contractual controls, retention limits, encryption. Example
  detection and containment: prompt monitoring, retrieval audit trails,
  suspicious tool-call detection, tool-call logging, an agent kill switch,
  token revocation, vendor incident escalation, integration disablement.
- **"A policy is not automatically an effective barrier."** Evidence is
  needed that the control actually prevents, detects or limits the failure.
- **Four questions per barrier:** design (does it address the specific
  leakage mechanism), coverage (is it deployed across all relevant
  applications, users, agents and data sources), effectiveness (has it been
  tested against realistic scenarios), evidence (can it be shown to work
  using logs, test results, configurations or audit records). The example
  given: DLP on employee prompts but nothing equivalent on AI agent tool
  calls — the control exists and a major pathway remains open.
- **Enterprise risk management attributes** for every pathway: risk owner,
  control owner, inherent risk (likelihood and impact before controls),
  control maturity (designed, deployed, tested, monitored), residual risk,
  evidence, gaps (missing or ineffective barriers), and action (remediation,
  owner, due date).
- **The recommended next drill-down**: the RAG data leakage pathway, as a
  sub-bow-tie whose top event is "An enterprise AI assistant discloses
  restricted information to an unauthorised user", tracing inherited
  document permissions, stale access-control lists, overly broad retrieval,
  cross-user memory, prompt injection and insecure connectors. The stated
  purpose is to map individual controls to individual failure mechanisms,
  identify common-cause control failures, and measure whether the
  enterprise is actually protected.

**What this means for the module.**
- The consequences side was only drawn in an image the page text does not
  carry. The consequences in the shipped AI safety material are therefore
  **illustrative**, written for the examples and tests, and not taken from
  the conversation.
- The four questions and the ERM attributes map directly to `barrier.design`,
  `barrier.coverage`, `barrier.maturity` plus `evidence`, and to `owner`,
  `assessment`, `action` and the derived gaps.
- The RAG sub-bow-tie is why a risk can name a `parent_pathway_id`.
- "Common-cause control failures" is why a control is shared across barriers
  and why `Reliance` exists.

## Standards the design leans on

Named from general knowledge, not quoted. No clause numbers or wording are
reproduced here and none were checked while writing this page; check them
before citing.

- **ISO 31000**, the general guidance on risk management: risk owners,
  treatment, monitoring and review, and the idea of a risk register.
- **ISO/IEC 31010**, on risk assessment techniques, which as far as the
  author recalls includes bow-tie analysis among them. The bow-tie method
  itself — hazard, top event, threats, consequences, preventive and
  mitigating barriers — is the structure the owner's conversation used.

Neither standard is a requirement of this module; the module is a
data model and a set of questions, not a compliance claim.
