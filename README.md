# Job Application Notes Tracker

A Go web app to help store and find job-application notes and artifacts.

## Features

- **Hybrid storage**: Filesystem for notes/artifacts (easy to read/edit), SQLite for metadata and search
- **Server-rendered HTML UI**: Browse companies, roles, threads, and meetings
- **Role status tracking**: Enum-based role status (recruiter_reached_out, hr_interview, etc.)
- **Computed company status**: Automatically computed from role statuses (in_progress, offer, rejected)
- **Deterministic export**: `export.json` for readable diffs and backup

## Quick Start

```bash
# Run the server
go run ./cmd/server

# Server starts at http://127.0.0.1:8080
```

Open your browser to `http://127.0.0.1:8080/companies` to see the UI.

## Configuration

Set environment variables to customize:

| Variable | Default | Description |
| ---------- | --------- | ------------- |
| `JOBTRACKER_REPO_ROOT` | Current directory | Root directory for data files |
| `JOBTRACKER_DB_PATH` | `db/index.sqlite` | Path to SQLite database |
| `JOBTRACKER_ADDR` | `127.0.0.1:8080` | Server bind address |

### Access from other devices

By default, the server only accepts local connections. To access from other devices on your network:

```bash
JOBTRACKER_ADDR=0.0.0.0:8080 go run ./cmd/server
```

Then access via your machine's IP (e.g., `http://192.168.0.103:8080`).

## Usage

### HTML Pages

| URL | Description |
| ----- | ------------- |
| `/companies` | List companies + "Add Company" form |
| `/companies/{slug}` | Company detail: roles, meetings + "Add Role" and "Add Meeting" forms |
| `/companies/{slug}/roles/{roleSlug}` | Role detail + "Attach JD" form |
| `/threads` | List threads + "Add Contact" and "Add Thread" forms |
| `/threads/{id}` | Thread detail: linked roles, meetings + "Link Role" and "Add Meeting" forms |

The nav bar includes:

- Links to Companies and Threads
- "Export JSON" button to generate `db/export.json`

### API Endpoints

All data is managed via the JSON API:

#### Companies & Roles

```bash
# Create a company
curl -X POST http://localhost:8080/api/companies \
  -H "Content-Type: application/json" \
  -d '{"slug": "acme-corp", "name": "Acme Corporation"}'

# List companies
curl http://localhost:8080/api/companies

# Get company details (with roles and meetings)
curl http://localhost:8080/api/companies/acme-corp

# Create a role under a company
curl -X POST http://localhost:8080/api/companies/acme-corp/roles \
  -H "Content-Type: application/json" \
  -d '{"slug": "senior-engineer", "title": "Senior Software Engineer"}'
```

#### Contacts & Threads

```bash
# Create a contact
curl -X POST http://localhost:8080/api/contacts \
  -H "Content-Type: application/json" \
  -d '{"name": "Jane Recruiter", "email": "jane@acme.com", "org": "Acme Corp"}'

# Create a thread (conversation container)
curl -X POST http://localhost:8080/api/threads \
  -H "Content-Type: application/json" \
  -d '{"title": "Acme Outreach", "contact_id": "<contact-id>"}'

# Get thread details
curl http://localhost:8080/api/threads/<thread-id>

# Link a thread to a role (can link to multiple roles across companies)
curl -X POST http://localhost:8080/api/threads/<thread-id>/roles \
  -H "Content-Type: application/json" \
  -d '{"role_ref": "acme-corp/senior-engineer"}'
```

#### Meetings

```bash
# Create a meeting (creates a markdown note file)
curl -X POST http://localhost:8080/api/meetings \
  -H "Content-Type: application/json" \
  -d '{
    "company_slug": "acme-corp",
    "thread_id": "<thread-id>",
    "occurred_at": "2024-01-15T10:00:00Z",
    "title": "Phone Screen"
  }'
```

#### Job Descriptions

```bash
# Attach JD (HTML and/or PDF)
curl -X POST http://localhost:8080/api/roles/acme-corp/senior-engineer/jd \
  -F "html=<html><body>Job description...</body></html>" \
  -F "pdf=@job-description.pdf"
```

#### Export

```bash
# Export all data to db/export.json
curl -X POST http://localhost:8080/api/export
```

## Data Storage

Data is stored in two places:

### Filesystem (`data/` directory)

```tree
data/
└── companies/
    └── acme-corp/
        ├── company.md          # Company notes (status is computed from roles)
        ├── meetings/
        │   └── 2024-01-15-phone-screen.md
        └── roles/
            └── senior-engineer/
                ├── job.html    # Job description HTML
                └── job.pdf     # Job description PDF
```

### SQLite (`db/index.sqlite`)

Stores metadata, relationships, and enables search:

- Companies, roles, contacts, threads, meetings
- Thread-role links, meeting-thread links
- Job description paths

## Role Status Management

Role status is managed via the UI on the role detail page (`/companies/{slug}/roles/{roleSlug}`).

### Available Role Statuses

| Status | Description |
| -------- | ------------- |
| `recruiter_reached_out` | Initial contact from recruiter (default) |
| `hr_interview` | HR/screening interview stage |
| `pairing_interview` | Technical pairing interview |
| `take_home_assignment` | Take-home assignment stage |
| `design_interview` | System design interview |
| `in_progress` | Generic in-progress stage |
| `offer` | Received an offer (terminal) |
| `rejected` | Rejected (terminal) |

### Computed Company Status

Company status is automatically computed from its roles:

- **in_progress**: Any role is not terminal (not rejected/offer)
- **offer**: All roles are terminal, and at least one is offer
- **rejected**: All roles are rejected

The computed status appears on the companies list and company detail pages.

### API for Status Updates

```bash
# Update role status
curl -X PATCH http://localhost:8080/api/companies/acme-corp/roles/senior-engineer/status \
  -H "Content-Type: application/json" \
  -d '{"status": "hr_interview"}'
```

## Testing

```bash
go test ./...
```

## Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```
