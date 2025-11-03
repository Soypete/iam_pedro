# Meetup Mode Implementation Plan

## GitHub Issues

**Epic Issue**: [#73 - Add Meetup Mode for Forge Utah Foundation Events](https://github.com/Soypete/iam_pedro/issues/73)

**Implementation Phases**:
- [#74 - Phase 1 MVP: Basic Meetup Mode with Prompt Injection](https://github.com/Soypete/iam_pedro/issues/74) ⚡ **START HERE**
- [#75 - Phase 2: Multi-Meetup Support and Config Templates](https://github.com/Soypete/iam_pedro/issues/75)
- [#76 - Phase 3: Database Schema for Semantic Search](https://github.com/Soypete/iam_pedro/issues/76)
- [#77 - Phase 4: Semantic Search Integration](https://github.com/Soypete/iam_pedro/issues/77)
- [#78 - Phase 5 (Optional): Management Tools and Monitoring](https://github.com/Soypete/iam_pedro/issues/78)

**Future Enhancements**:
- [#79 - Add Meetup Mode support to Discord bot](https://github.com/Soypete/iam_pedro/issues/79) (Reuse vector store across platforms)

## Overview

Add a configurable "Meetup Mode" to Pedro (Twitch bot) that enhances responses with information about Forge Utah Foundation meetups. This feature allows Pedro to provide context about current/upcoming events, speakers, schedules, and community information during streams.

## Background

- **Forge Utah Foundation**: Non-profit supporting local tech communities
- **Use Case**: When streaming meetup events, Pedro should help viewers learn about the event, speakers, schedule, and how to participate
- **Pattern**: Similar to the removed GoWest conference mode - inject meetup context into Pedro's system prompt

## Architecture Decision: Data Source Strategy

### Recommended: Three-Tier Hybrid Approach

#### Tier 1: Static Prompt Injection (PRIMARY - MVP)
- **Method**: Inject meetup info into system prompt addendum
- **Source**: YAML config files in `/configs/meetups/`
- **Best For**: Event schedule, location, registration, general info
- **Advantages**:
  - Fastest response time
  - No external dependencies
  - Works offline
  - LLM has full context
  - Proven pattern (GoWest mode)

#### Tier 2: Semantic Search with Vector Embeddings (FUTURE)
- **Method**: PostgreSQL vector search using existing `POSTGRES_VECTOR_URL`
- **Source**: Pre-indexed speaker bios, FAQs, community info
- **Best For**: "Tell me about [speaker]", "How do I get involved?", nuanced questions
- **Advantages**:
  - Handles complex queries
  - Scales to large content libraries
  - Fast (local database)
  - Full control over information

#### Tier 3: Web Search (FALLBACK ONLY)
- **Method**: Existing DuckDuckGo integration
- **Best For**: Real-time updates, external information
- **When to Use**: If Tiers 1 & 2 can't answer

### Why NOT Web Search as Primary?
- Forge Utah content may not be well-indexed
- Latency issues (async responses)
- Less reliable for niche non-profit info
- We already have better solutions (prompt + vector DB)

---

## Implementation Phases

### Phase 1: MVP - Basic Meetup Mode ✅ **Start Here**
**Goal**: Get working for this week's meetup stream

**Changes Required**:
1. Add CLI flag: `-meetupMode string` (e.g., "golang-nov-2025")
2. Create meetup config YAML structure
3. Add meetup prompt addendum to `ai/chatter.go`
4. Modify `ai/twitchchat/llm.go` to inject addendum
5. Create config for November Go meetup

**Files to Modify**:
- `cli/twitch/twitch.go` - Add CLI flag
- `ai/chatter.go` - Add MeetupAddendum variable/loader
- `ai/twitchchat/client.go` - Add meetupMode field
- `ai/twitchchat/llm.go` - Modify `callLLM()` to inject addendum

**Files to Create**:
- `configs/meetups/golang-nov-2025.yaml` - First meetup config
- `ai/meetup_config.go` - Config loader

**Estimated Time**: 4-6 hours

---

### Phase 2: Multiple Meetup Support
**Goal**: Support all Forge Utah meetup groups

**Features**:
- Config file validation
- Multiple meetup templates
- Meetup config management tools
- Documentation for adding new meetups

**Files to Create**:
- `configs/meetups/golang-template.yaml`
- `configs/meetups/data-template.yaml`
- `configs/meetups/README.md` - How to add meetups

**Estimated Time**: 3-4 hours

---

### Phase 3: Database Schema for Semantic Search
**Goal**: Enable rich speaker/FAQ queries

**Database Tables**:
```sql
-- Core meetup events
CREATE TABLE forge_meetups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    event_date TIMESTAMP NOT NULL,
    location VARCHAR(255),
    registration_url VARCHAR(500),
    video_call_link VARCHAR(500),
    meetup_group VARCHAR(100), -- e.g., "golang", "data"
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Speaker information with embeddings
CREATE TABLE meetup_speakers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meetup_id UUID REFERENCES forge_meetups(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    bio TEXT,
    bio_embedding VECTOR(1536), -- For semantic search
    talk_title VARCHAR(500),
    talk_description TEXT,
    social_links JSONB, -- GitHub, Twitter, LinkedIn
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_speaker_embedding ON meetup_speakers
USING ivfflat (bio_embedding vector_cosine_ops);

-- FAQ with embeddings for semantic matching
CREATE TABLE meetup_faqs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meetup_id UUID REFERENCES forge_meetups(id) ON DELETE SET NULL,
    category VARCHAR(100) NOT NULL, -- "general", "forge", "logistics", "speaking"
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    question_embedding VECTOR(1536),
    is_global BOOLEAN DEFAULT false, -- True for Forge-wide FAQs
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_faq_embedding ON meetup_faqs
USING ivfflat (question_embedding vector_cosine_ops);

-- Forge Utah Foundation info
CREATE TABLE forge_info (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    info_type VARCHAR(100) NOT NULL, -- "mission", "contact", "sponsorship"
    content TEXT NOT NULL,
    content_embedding VECTOR(1536),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**Files to Create**:
- `database/migrations/YYYYMMDDHHMMSS_create_meetup_tables.sql`
- `database/meetup_reader.go` - Query interface
- `database/meetup_writer.go` - Insert/update interface

**Estimated Time**: 6-8 hours

---

### Phase 4: Semantic Search Integration
**Goal**: Enable smart Q&A about speakers and Forge Utah

**Features**:
- Generate embeddings for speaker bios
- Generate embeddings for FAQs
- Semantic search functions
- Integration with LLM flow
- Question classification (speaker vs FAQ vs general)

**Files to Create**:
- `ai/embeddings.go` - Embedding generation helpers
- `ai/twitchchat/semantic_search.go` - Search logic

**Files to Modify**:
- `ai/twitchchat/llm.go` - Add semantic search before LLM call

**Flow**:
1. User asks: "Who is Miriah Peterson?"
2. Detect speaker question pattern
3. Query `meetup_speakers` with semantic search
4. Inject speaker bio into prompt
5. Generate informed response

**Estimated Time**: 8-10 hours

---

### Phase 5: Management Tools (Optional)
**Goal**: Easy meetup content management

**Features**:
- CLI tool to add/update meetups
- Bulk FAQ import from CSV
- Speaker bio management
- Embedding regeneration

**Files to Create**:
- `cli/meetup-admin/main.go` - Admin CLI
- `scripts/import_faqs.go`

**Estimated Time**: 4-6 hours

---

## Meetup Config File Format

### Example: `configs/meetups/golang-nov-2025.yaml`

```yaml
metadata:
  slug: "golang-nov-2025"
  name: "Utah Go User Group - November 2025"
  meetup_group: "golang"
  date: "2025-11-06T18:30:00-07:00"  # MST
  duration_minutes: 60

event_info:
  title: "Beyond Hello World: Mastering net/http for Production Go Services"
  description: "A practical deep dive into Go's net/http package covering building robust HTTP servers, request routing patterns, authentication strategies, connection pooling configuration, and testing with httptest."

location:
  type: "virtual"  # virtual, hybrid, in-person
  venue: "Google Meet"
  address: ""
  parking: ""
  accessibility: ""

links:
  registration: "https://www.meetup.com/utah-go-user-group/events/..."
  video_call: "https://meet.google.com/dkn-ekxb-qrj"
  dial_in: "+1 336-673-3641, PIN: 540 425 241#"

schedule:
  - time: "18:30"
    event: "Introduction and announcements"
    duration_minutes: 5
  - time: "18:35"
    event: "Main Presentation: Beyond Hello World"
    duration_minutes: 40
    speaker: "Miriah Peterson"
  - time: "19:15"
    event: "Open discussion - tech/architecture/ops"
    duration_minutes: 45

speakers:
  - name: "Miriah Peterson"
    title: "Member of Technical Staff"
    handle: "SoyPeteTech"
    bio: "Self-taught developer based in Utah, specializing in Go and Data/AI. Streams live coding on Twitch."
    talk_title: "Beyond Hello World: Mastering net/http for Production Go Services"
    talk_description: "Learn why you don't always need a framework and how to avoid common pitfalls like using the default HTTP client in production."
    social:
      github: "soypete"
      twitch: "soypetetech"
      twitter: ""

hosts:
  - "Clint B."
  - "Miriah P."

forge_info:
  is_forge_event: true
  about: "Utah Go User Group is part of Forge Utah Foundation, a non-profit supporting local tech communities through meetups, workshops, and networking."
  other_meetups:
    - "Data Engineering & AI Meetup"
    - "Cloud Native Utah"
  community_links:
    website: "https://forgeutah.org"
    slack: ""
    discord: ""
  how_to_join: "Visit forgeutah.org or join us on Meetup.com"
  how_to_speak: "Reach out to organizers on Meetup or ping us in community chat"

sponsor:
  name: ""
  message: ""
  logo_url: ""

bot_instructions:
  response_style: "enthusiastic and welcoming"
  key_points:
    - "This is a virtual event - anyone can join via Google Meet"
    - "Great for Go developers at all levels"
    - "Part of Forge Utah's mission to build tech community"
    - "Encourage questions during the open discussion portion"
  trigger_phrases:
    - "tell me about the meetup"
    - "what's tonight about"
    - "who is speaking"
    - "how do I join"
    - "what is forge utah"
```

---

## Prompt Addendum Template

When `meetupMode` is enabled, append to `PedroPrompt`:

```
SPECIAL EVENT - [Event Title] ([Date at Time]):
We're streaming/discussing the [Meetup Group Name] meetup tonight!

Event: [Title]
Speaker: [Name] - [Talk Title]
When: [Date] at [Time] [Timezone]
Where: [Virtual/Location]
Join: [Video Call Link]

Schedule:
[Time] - [Event 1]
[Time] - [Event 2]
[Time] - [Event 3]

About the Speaker:
[Speaker Bio - 2-3 sentences]

Forge Utah Foundation:
This meetup is part of Forge Utah Foundation, a non-profit supporting Utah's tech community through meetups, workshops, and networking. We run [list other meetups]. Anyone can participate - check out forgeutah.org to learn more or join future events!

How to Get Involved:
- Attend meetups (free!)
- Propose talk ideas to organizers
- Join our community: [links]
- Support as sponsor: [contact]

When viewers ask about:
- The event: Share title, speaker, time, and registration link enthusiastically
- Forge Utah: Explain our mission and other meetups
- Speaking opportunities: Encourage them to reach out to organizers
- The speaker: Share bio and talk topic

Be welcoming to newcomers and encourage participation!
```

---

## FAQ Database Content (For Phase 4)

### General Forge Utah FAQs
```yaml
faqs:
  - category: "general"
    question: "What is Forge Utah Foundation?"
    answer: "Forge Utah Foundation is a non-profit organization supporting local tech communities in Utah through meetups, workshops, and networking events. We organize regular Go, Data Engineering, and Cloud Native meetups."

  - category: "general"
    question: "How do I join Forge Utah?"
    answer: "Anyone can participate! Follow our meetups on Meetup.com, join our Slack/Discord, or visit forgeutah.org. All events are free and open to the community."

  - category: "speaking"
    question: "How do I speak at a Forge Utah meetup?"
    answer: "We welcome speakers! Reach out to the organizers on the Meetup event page, ping us in community chat, or email [contact]. Topics can range from beginner to advanced - we value all perspectives."

  - category: "logistics"
    question: "Are meetups virtual or in-person?"
    answer: "It varies by event! Check each meetup posting. Many are hybrid or virtual to maximize accessibility. Location and video call details are always in the event description."

  - category: "sponsorship"
    question: "How do I sponsor a Forge Utah meetup?"
    answer: "We appreciate sponsors! Contact us at [email] or reach out to organizers on Meetup. Sponsorships help us provide food, venue, and resources for the community."
```

---

## Testing Plan

### Manual Testing Checklist
- [ ] Bot starts with `-meetupMode golang-nov-2025`
- [ ] Pedro responds to "what's the meetup tonight?"
- [ ] Pedro shares speaker info when asked
- [ ] Pedro provides registration link
- [ ] Pedro explains Forge Utah when asked
- [ ] Pedro directs people to community resources
- [ ] Prompt stays under character limit
- [ ] Responses are enthusiastic and welcoming
- [ ] Bot works normally when meetupMode disabled

### Test Scenarios
```
User: "pedro what's tonight about?"
Expected: Info about net/http talk, speaker, time

User: "who is speaking?"
Expected: Miriah Peterson bio, talk title

User: "how do I join the meetup?"
Expected: Video call link + registration encouragement

User: "what is forge utah?"
Expected: Mission, other meetups, how to get involved

User: "can I speak at a future meetup?"
Expected: Yes! Here's how to reach organizers
```

---

## Deployment Notes

### Environment Variables (Existing)
```bash
LLAMA_CPP_PATH="http://127.0.0.1:8080"
POSTGRES_URL="postgres://..."
POSTGRES_VECTOR_URL="postgres://..."  # For Phase 4
TWITCH_ID="..."
TWITCH_SECRET="..."
```

### CLI Usage
```bash
# With meetup mode enabled
go run ./cli/twitch \
  -model "your-model-name" \
  -meetupMode "golang-nov-2025" \
  -errorLevel debug

# Normal mode (default)
go run ./cli/twitch \
  -model "your-model-name" \
  -errorLevel debug
```

### Docker Build (Future)
```dockerfile
# Copy meetup configs into container
COPY configs/meetups /app/configs/meetups

# Run with meetup mode
CMD ["/app/twitch", "-model", "${MODEL_NAME}", "-meetupMode", "${MEETUP_SLUG}"]
```

---

## Success Metrics

### Phase 1 Success Criteria
- Bot successfully injects meetup context
- Pedro answers meetup questions accurately
- Stream viewers engage with meetup info
- No performance degradation
- Easy to enable/disable

### Phase 4 Success Criteria (Future)
- Semantic search returns relevant speaker/FAQ info
- Response time < 2 seconds for embedded searches
- 80%+ accuracy on meetup-related questions
- Easy content management workflow

---

## Future Enhancements

1. **Calendar Integration**: Auto-load upcoming meetups from Meetup.com API
2. **Multi-Platform**: Add Discord meetup mode
3. **Metrics**: Track meetup-related question volume
4. **Attendee Management**: Integration with registration systems
5. **Post-Event**: Automatically share recordings/slides
6. **Meetup Discovery**: "What other Forge meetups should I check out?"

---

## References

- Previous Implementation: GoWest mode (commit `bee6892`)
  - Pattern: Boolean flag + prompt addendum
  - File: `ai/chatter.go` (GoWestAddendum)
  - Integration: `ai/twitchchat/llm.go` (callLLM function)
- Existing Features to Leverage:
  - Web search: `ai/twitchchat/llm.go:130-185`
  - Vector DB: `POSTGRES_VECTOR_URL` environment variable
  - Config pattern: CLI flags in `cli/twitch/twitch.go`

---

## Questions & Decisions

### Resolved
- ✅ Use semantic search over web search as primary
- ✅ Store configs as YAML files
- ✅ Follow GoWest pattern for MVP
- ✅ Use three-tier data strategy

### Open Questions
- Which embedding model for semantic search? (langchain-go supports multiple)
- Hosting: Where to run embedding generation? (Could be local CLI tool)
- Content management: Who updates meetup configs? (Git-based workflow vs admin UI)
- Meetup.com API: Should we auto-sync event details?

---

## Timeline Estimate

- **Phase 1 (MVP)**: 1 week - **TARGET FOR THIS WEEK'S STREAM** ⚡
- **Phase 2**: 3-4 days
- **Phase 3**: 1 week
- **Phase 4**: 1.5 weeks
- **Phase 5**: 3-4 days (optional)

**Total**: ~4-5 weeks for full implementation

---

## Next Steps

1. Create GitHub issues (see GITHUB_ISSUES.md)
2. Create feature branch: `feature/meetup-mode-forge-utah`
3. Implement Phase 1 MVP for Nov meetup
4. Test during this week's stream
5. Gather feedback and iterate
