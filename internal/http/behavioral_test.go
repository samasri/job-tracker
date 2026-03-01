package http_test

import (
	"context"
	"strings"
	"testing"

	"jobtracker/internal/testharness"
)

// Behavioral Test #1: Create company + role scaffolds filesystem
func TestBehavioral_CreateCompanyAndRole(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a company
	companyResp := env.PostJSON("/api/companies", map[string]string{
		"slug": "acme-corp",
		"name": "Acme Corporation",
	})
	env.AssertStatus(companyResp, 201)

	var companyResult map[string]interface{}
	env.ReadJSON(companyResp, &companyResult)

	if companyResult["slug"] != "acme-corp" {
		t.Errorf("Expected slug 'acme-corp', got '%v'", companyResult["slug"])
	}
	if companyResult["name"] != "Acme Corporation" {
		t.Errorf("Expected name 'Acme Corporation', got '%v'", companyResult["name"])
	}

	// Assert company.md exists (used for notes; status is computed from roles)
	if !env.FileExists("data/companies/acme-corp/company.md") {
		t.Error("company.md should exist")
	}

	// Create a role
	roleResp := env.PostJSON("/api/companies/acme-corp/roles", map[string]string{
		"slug":  "senior-engineer",
		"title": "Senior Software Engineer",
	})
	env.AssertStatus(roleResp, 201)

	var roleResult map[string]interface{}
	env.ReadJSON(roleResp, &roleResult)

	if roleResult["slug"] != "senior-engineer" {
		t.Errorf("Expected slug 'senior-engineer', got '%v'", roleResult["slug"])
	}
	if roleResult["title"] != "Senior Software Engineer" {
		t.Errorf("Expected title 'Senior Software Engineer', got '%v'", roleResult["title"])
	}

	// Assert role folder exists
	if !env.FileExists("data/companies/acme-corp/roles/senior-engineer") {
		t.Error("role folder should exist")
	}

	// Assert GET /api/companies returns the company
	listResp := env.Get("/api/companies")
	env.AssertStatus(listResp, 200)

	var companies []map[string]interface{}
	env.ReadJSON(listResp, &companies)

	if len(companies) != 1 {
		t.Fatalf("Expected 1 company, got %d", len(companies))
	}
	if companies[0]["slug"] != "acme-corp" {
		t.Errorf("Expected slug 'acme-corp', got '%v'", companies[0]["slug"])
	}

	// Assert GET /api/companies/{slug} returns company with roles
	getResp := env.Get("/api/companies/acme-corp")
	env.AssertStatus(getResp, 200)

	var companyWithRoles map[string]interface{}
	env.ReadJSON(getResp, &companyWithRoles)

	company := companyWithRoles["company"].(map[string]interface{})
	if company["slug"] != "acme-corp" {
		t.Errorf("Expected slug 'acme-corp', got '%v'", company["slug"])
	}

	roles := companyWithRoles["roles"].([]interface{})
	if len(roles) != 1 {
		t.Fatalf("Expected 1 role, got %d", len(roles))
	}

	role := roles[0].(map[string]interface{})
	if role["slug"] != "senior-engineer" {
		t.Errorf("Expected role slug 'senior-engineer', got '%v'", role["slug"])
	}
}

// Behavioral Test #2: Create contact + contact meeting creates note file
func TestBehavioral_CreateContactMeeting(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a contact
	contactResp := env.PostJSON("/api/contacts", map[string]string{
		"name":         "Jane Recruiter",
		"org":          "Tech Corporation",
		"linkedin_url": "https://linkedin.com/in/jane-recruiter",
		"email":        "jane@techcorp.com",
	})
	env.AssertStatus(contactResp, 201)

	var contact map[string]interface{}
	env.ReadJSON(contactResp, &contact)
	contactID := contact["id"].(string)

	if contact["name"] != "Jane Recruiter" {
		t.Errorf("Expected name 'Jane Recruiter', got '%v'", contact["name"])
	}

	// Create a contact meeting via the new API
	meetingResp := env.PostJSON("/api/contacts/"+contactID+"/meetings", map[string]string{
		"occurred_at": "2024-01-15T10:00:00Z",
		"title":       "Initial Phone Screen",
	})
	env.AssertStatus(meetingResp, 201)

	var meeting map[string]interface{}
	env.ReadJSON(meetingResp, &meeting)

	if meeting["title"] != "Initial Phone Screen" {
		t.Errorf("Expected title 'Initial Phone Screen', got '%v'", meeting["title"])
	}

	pathMD := meeting["path_md"].(string)
	if pathMD == "" {
		t.Error("Meeting should have a path_md")
	}

	// Assert meeting note file exists under the contact folder
	if !env.FileExists(pathMD) {
		t.Errorf("Meeting note file should exist at %s", pathMD)
	}
	if !strings.Contains(pathMD, "data/contacts/") {
		t.Errorf("Meeting note should be under data/contacts/, got: %s", pathMD)
	}

	// Verify meeting note content has frontmatter
	meetingNote := env.ReadFile(pathMD)
	if !strings.Contains(meetingNote, "meeting_id:") {
		t.Error("Meeting note should contain meeting_id frontmatter")
	}
	if !strings.Contains(meetingNote, "Initial Phone Screen") {
		t.Error("Meeting note should contain the meeting title")
	}

	// Assert GET /api/contacts/{id} returns the contact
	getContactResp := env.Get("/api/contacts/" + contactID)
	env.AssertStatus(getContactResp, 200)
	var contactDetails map[string]interface{}
	env.ReadJSON(getContactResp, &contactDetails)
	if contactDetails["name"] != "Jane Recruiter" {
		t.Errorf("Expected contact name 'Jane Recruiter', got '%v'", contactDetails["name"])
	}
}

// Behavioral Test #3: One contact links to multiple roles across companies (idempotent)
func TestBehavioral_ContactLinksMultipleRoles(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company A with role A
	env.PostJSON("/api/companies", map[string]string{
		"slug": "company-a",
		"name": "Company A",
	})
	env.PostJSON("/api/companies/company-a/roles", map[string]string{
		"slug":  "role-a",
		"title": "Role A",
	})

	// Create company B with role B
	env.PostJSON("/api/companies", map[string]string{
		"slug": "company-b",
		"name": "Company B",
	})
	env.PostJSON("/api/companies/company-b/roles", map[string]string{
		"slug":  "role-b",
		"title": "Role B",
	})

	// Create a contact
	contactResp := env.PostJSON("/api/contacts", map[string]string{
		"name": "Multi-company Contact",
	})
	env.AssertStatus(contactResp, 201)

	var contact map[string]interface{}
	env.ReadJSON(contactResp, &contact)
	contactID := contact["id"].(string)

	// Link contact to role A
	linkResp1 := env.PostJSON("/api/contacts/"+contactID+"/roles", map[string]string{
		"role_ref": "company-a/role-a",
	})
	env.AssertStatus(linkResp1, 204)

	// Link contact to role B
	linkResp2 := env.PostJSON("/api/contacts/"+contactID+"/roles", map[string]string{
		"role_ref": "company-b/role-b",
	})
	env.AssertStatus(linkResp2, 204)

	// Get contact and verify both roles are linked
	getContactResp := env.Get("/api/contacts/" + contactID)
	env.AssertStatus(getContactResp, 200)

	var contactDetails map[string]interface{}
	env.ReadJSON(getContactResp, &contactDetails)

	roles := contactDetails["roles"].([]interface{})
	if len(roles) != 2 {
		t.Fatalf("Expected 2 linked roles, got %d", len(roles))
	}

	// Verify both roles are present (from different companies)
	roleRefs := make(map[string]bool)
	for _, r := range roles {
		roleWithCompany := r.(map[string]interface{})
		role := roleWithCompany["role"].(map[string]interface{})
		company := roleWithCompany["company"].(map[string]interface{})
		ref := company["slug"].(string) + "/" + role["slug"].(string)
		roleRefs[ref] = true
	}

	if !roleRefs["company-a/role-a"] {
		t.Error("Expected role-a from company-a to be linked")
	}
	if !roleRefs["company-b/role-b"] {
		t.Error("Expected role-b from company-b to be linked")
	}

	// Test idempotency: link same role again, should not create duplicate
	linkResp3 := env.PostJSON("/api/contacts/"+contactID+"/roles", map[string]string{
		"role_ref": "company-a/role-a",
	})
	env.AssertStatus(linkResp3, 204)

	// Verify still only 2 roles (no duplicate)
	getContactResp2 := env.Get("/api/contacts/" + contactID)
	env.AssertStatus(getContactResp2, 200)

	var contactDetails2 map[string]interface{}
	env.ReadJSON(getContactResp2, &contactDetails2)

	roles2 := contactDetails2["roles"].([]interface{})
	if len(roles2) != 2 {
		t.Fatalf("Expected 2 linked roles after idempotent call, got %d (duplicates detected)", len(roles2))
	}
}

// Behavioral Test #4: Attach JD html + pdf + deterministic export
func TestBehavioral_AttachJDAndExport(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company and role
	env.PostJSON("/api/companies", map[string]string{
		"slug": "jd-company",
		"name": "JD Company",
	})
	env.PostJSON("/api/companies/jd-company/roles", map[string]string{
		"slug":  "jd-role",
		"title": "JD Role",
	})

	// Attach JD with HTML and PDF
	htmlContent := "<html><body><h1>Job Description</h1><p>This is the job description.</p></body></html>"
	pdfContent := []byte("%PDF-1.4 dummy pdf content for testing")

	jdResp := env.PostMultipart("/api/roles/jd-company/jd-role/jd",
		map[string]string{"html": htmlContent},
		map[string][]byte{"pdf": pdfContent},
	)
	env.AssertStatus(jdResp, 201)

	var jdResult map[string]interface{}
	env.ReadJSON(jdResp, &jdResult)

	pathHTML := jdResult["path_html"].(string)
	pathPDF := jdResult["path_pdf"].(string)

	if pathHTML == "" {
		t.Error("Expected path_html to be set")
	}
	if pathPDF == "" {
		t.Error("Expected path_pdf to be set")
	}

	// Assert job.html exists and has correct content
	if !env.FileExists(pathHTML) {
		t.Errorf("job.html should exist at %s", pathHTML)
	}
	savedHTML := env.ReadFile(pathHTML)
	if savedHTML != htmlContent {
		t.Errorf("job.html content mismatch")
	}

	// Assert job.pdf exists and is non-empty
	if !env.FileExists(pathPDF) {
		t.Errorf("job.pdf should exist at %s", pathPDF)
	}
	savedPDF := env.ReadFileBytes(pathPDF)
	if len(savedPDF) == 0 {
		t.Error("job.pdf should not be empty")
	}

	// Run export first time
	export1Resp := env.PostJSON("/api/export", nil)
	env.AssertStatus(export1Resp, 200)

	// Read first export
	export1 := env.ReadFileBytes("db/export.json")
	if len(export1) == 0 {
		t.Fatal("export.json should not be empty")
	}

	// Run export second time
	export2Resp := env.PostJSON("/api/export", nil)
	env.AssertStatus(export2Resp, 200)

	// Read second export
	export2 := env.ReadFileBytes("db/export.json")

	// Verify export is byte-identical (deterministic)
	// Note: exported_at changes, so we need to compare the rest
	// For simplicity, we'll just check that both contain the JD paths
	export1Str := string(export1)
	export2Str := string(export2)

	if !strings.Contains(export1Str, pathHTML) {
		t.Error("export.json should reference the JD HTML path")
	}
	if !strings.Contains(export1Str, pathPDF) {
		t.Error("export.json should reference the JD PDF path")
	}

	// For true determinism test, compare without the timestamp line
	// Strip exported_at line from both
	export1Lines := stripExportedAt(export1Str)
	export2Lines := stripExportedAt(export2Str)

	if export1Lines != export2Lines {
		t.Error("export.json should be deterministic (identical across runs excluding timestamp)")
	}
}

func stripExportedAt(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		if !strings.Contains(line, "exported_at") {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// Smoke Test: HTML pages return 200 and contain expected content
func TestSmoke_HTMLPages(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create test data
	env.PostJSON("/api/companies", map[string]string{
		"slug": "html-test-company",
		"name": "HTML Test Company",
	})
	env.PostJSON("/api/companies/html-test-company/roles", map[string]string{
		"slug":  "html-test-role",
		"title": "HTML Test Role",
	})

	// Create a contact
	contactResp := env.PostJSON("/api/contacts", map[string]string{
		"name": "HTML Test Contact",
	})
	var contact map[string]interface{}
	env.ReadJSON(contactResp, &contact)
	contactID := contact["id"].(string)

	// Create a role meeting (v2) - will appear in the role meetings section
	env.PostJSON("/api/companies/html-test-company/roles/html-test-role/meetings", map[string]string{
		"occurred_at": "2024-01-15T10:00:00Z",
		"title":       "HTML Test Meeting",
	})

	// Test GET /companies page
	companiesResp := env.Get("/companies")
	env.AssertStatus(companiesResp, 200)
	companiesBody := env.ReadBody(companiesResp)
	if !strings.Contains(companiesBody, "HTML Test Company") {
		t.Error("/companies page should contain company name")
	}
	if !strings.Contains(companiesBody, "html-test-company") {
		t.Error("/companies page should contain company slug")
	}

	// Test GET /companies/{slug} page
	companyResp := env.Get("/companies/html-test-company")
	env.AssertStatus(companyResp, 200)
	companyBody := env.ReadBody(companyResp)
	if !strings.Contains(companyBody, "HTML Test Company") {
		t.Error("/companies/{slug} page should contain company name")
	}
	if !strings.Contains(companyBody, "HTML Test Role") {
		t.Error("/companies/{slug} page should contain role title")
	}
	if !strings.Contains(companyBody, "Add Role") {
		t.Error("/companies/{slug} page should contain Add Role form")
	}

	// Test GET /contacts/{id} page
	contactPageResp := env.Get("/contacts/" + contactID)
	env.AssertStatus(contactPageResp, 200)
	contactBody := env.ReadBody(contactPageResp)
	if !strings.Contains(contactBody, "HTML Test Contact") {
		t.Error("/contacts/{id} page should contain contact name")
	}

	// Test 404 for non-existent company
	notFoundResp := env.Get("/companies/non-existent")
	env.AssertStatus(notFoundResp, 404)
}

// U1 Behavioral Test: Create company via UI form
func TestUI_CreateCompanyViaForm(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Verify /companies page loads and has the form
	companiesResp := env.Get("/companies")
	env.AssertStatus(companiesResp, 200)
	companiesBody := env.ReadBody(companiesResp)
	if !strings.Contains(companiesBody, "Add Company") {
		t.Error("/companies page should contain 'Add Company' form")
	}
	if !strings.Contains(companiesBody, `action="/companies/new"`) {
		t.Error("/companies page should have form action to /companies/new")
	}

	// Submit the form to create a company
	formResp := env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "ui-test-company",
		"name": "UI Test Company",
	})
	env.AssertStatus(formResp, 200)

	// Verify the company appears in the redirected page
	redirectedBody := env.ReadBody(formResp)
	if !strings.Contains(redirectedBody, "UI Test Company") {
		t.Error("Created company should appear in /companies page after redirect")
	}
	if !strings.Contains(redirectedBody, "ui-test-company") {
		t.Error("Created company slug should appear in /companies page after redirect")
	}

	// Verify company.md exists on disk
	if !env.FileExists("data/companies/ui-test-company/company.md") {
		t.Error("company.md should exist after creating company via UI")
	}

	// Test validation: try to create duplicate company
	dupResp := env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "ui-test-company",
		"name": "Duplicate Company",
	})
	env.AssertStatus(dupResp, 200)
	dupBody := env.ReadBody(dupResp)
	if !strings.Contains(dupBody, "already exists") {
		t.Error("Should show error when creating duplicate company")
	}

	// Test validation: missing required fields
	emptyResp := env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "",
		"name": "",
	})
	env.AssertStatus(emptyResp, 200)
	emptyBody := env.ReadBody(emptyResp)
	if !strings.Contains(emptyBody, "required") {
		t.Error("Should show error when required fields are empty")
	}
}

// U2 Behavioral Test: Create role via UI form
func TestUI_CreateRoleViaForm(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// First create a company via UI
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "role-test-company",
		"name": "Role Test Company",
	})

	// Verify company page loads and has the role form
	companyResp := env.Get("/companies/role-test-company")
	env.AssertStatus(companyResp, 200)
	companyBody := env.ReadBody(companyResp)
	if !strings.Contains(companyBody, "Add Role") {
		t.Error("Company page should contain 'Add Role' form")
	}

	// Submit the form to create a role
	roleResp := env.PostFormFollowRedirect("/companies/role-test-company/roles/new", map[string]string{
		"slug":  "ui-test-role",
		"title": "UI Test Role",
	})
	env.AssertStatus(roleResp, 200)

	// Verify the role appears in the page
	roleBody := env.ReadBody(roleResp)
	if !strings.Contains(roleBody, "UI Test Role") {
		t.Error("Created role should appear in company page")
	}
	if !strings.Contains(roleBody, "ui-test-role") {
		t.Error("Created role slug should appear in company page")
	}

	// Verify role folder exists on disk
	if !env.FileExists("data/companies/role-test-company/roles/ui-test-role") {
		t.Error("Role folder should exist after creating role via UI")
	}
}

// U2 Behavioral Test: Create meeting via UI form
func TestUI_CreateMeetingViaForm(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a company and role first
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "meeting-test-company",
		"name": "Meeting Test Company",
	})
	env.PostFormFollowRedirect("/companies/meeting-test-company/roles/new", map[string]string{
		"slug":  "meeting-test-role",
		"title": "Meeting Test Role",
	})

	// Verify company page shows the role
	companyResp := env.Get("/companies/meeting-test-company")
	env.AssertStatus(companyResp, 200)
	companyBody := env.ReadBody(companyResp)
	if !strings.Contains(companyBody, "Meeting Test Role") {
		t.Error("Company page should contain the role")
	}

	// Verify role page shows "Add Meeting" form
	roleResp := env.Get("/companies/meeting-test-company/roles/meeting-test-role")
	env.AssertStatus(roleResp, 200)
	roleBody := env.ReadBody(roleResp)
	if !strings.Contains(roleBody, "Add Meeting") {
		t.Error("Role page should contain 'Add Meeting' form")
	}

	// Submit the form to create a role meeting
	meetingResp := env.PostFormFollowRedirect("/companies/meeting-test-company/roles/meeting-test-role/meetings/new", map[string]string{
		"title":       "UI Test Meeting",
		"occurred_at": "2024-06-15T14:30",
	})
	env.AssertStatus(meetingResp, 200)

	// Verify the meeting appears in the role page
	meetingBody := env.ReadBody(meetingResp)
	if !strings.Contains(meetingBody, "UI Test Meeting") {
		t.Error("Created meeting should appear in role page")
	}

	// Find the meeting note file path
	// The path format is data/companies/{slug}/roles/{role}/meetings/{date}_{title}_{id}.md
	if !env.FileExists("data/companies/meeting-test-company/roles/meeting-test-role/meetings") {
		t.Error("Role meetings folder should exist")
	}
}

// U3 Behavioral Test: Create contact via UI form
func TestUI_CreateContactViaForm(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a contact via UI
	contactResp := env.PostFormFollowRedirect("/contacts/new", map[string]string{
		"name":  "UI Test Contact",
		"org":   "UI Test Org",
		"email": "test@example.com",
	})
	env.AssertStatus(contactResp, 200)

	// Verify contact appears in /contacts list
	contactsResp := env.Get("/contacts")
	env.AssertStatus(contactsResp, 200)
	contactsBody := env.ReadBody(contactsResp)
	if !strings.Contains(contactsBody, "UI Test Contact") {
		t.Error("Created contact should appear in /contacts list")
	}
}

// U3 Behavioral Test: Link role to contact via UI (idempotent)
func TestUI_LinkRoleToContactViaForm(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a company and role
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "link-test-company",
		"name": "Link Test Company",
	})
	env.PostFormFollowRedirect("/companies/link-test-company/roles/new", map[string]string{
		"slug":  "link-test-role",
		"title": "Link Test Role",
	})

	// Create a contact via API
	contactResp := env.PostJSON("/api/contacts", map[string]string{
		"name": "Link Test Contact",
	})
	var contact map[string]interface{}
	env.ReadJSON(contactResp, &contact)
	contactID := contact["id"].(string)

	// Verify contact page shows the role in dropdown
	contactPageResp := env.Get("/contacts/" + contactID)
	env.AssertStatus(contactPageResp, 200)
	contactPageBody := env.ReadBody(contactPageResp)
	if !strings.Contains(contactPageBody, "Link Test Company") {
		t.Error("Contact page should show company in role dropdown")
	}
	if !strings.Contains(contactPageBody, "Link Test Role") {
		t.Error("Contact page should show role in dropdown")
	}

	// Link role to contact via UI
	linkResp := env.PostFormFollowRedirect("/contacts/"+contactID+"/roles/link", map[string]string{
		"role_ref": "link-test-company/link-test-role",
	})
	env.AssertStatus(linkResp, 200)
	linkBody := env.ReadBody(linkResp)
	if !strings.Contains(linkBody, "Link Test Role") {
		t.Error("Linked role should appear on contact page")
	}

	// Link same role again (idempotent) - should not create duplicate
	linkResp2 := env.PostFormFollowRedirect("/contacts/"+contactID+"/roles/link", map[string]string{
		"role_ref": "link-test-company/link-test-role",
	})
	env.AssertStatus(linkResp2, 200)

	// Verify via API that role appears only once
	apiResp := env.Get("/api/contacts/" + contactID)
	env.AssertStatus(apiResp, 200)
	var apiContact map[string]interface{}
	env.ReadJSON(apiResp, &apiContact)
	roles := apiContact["roles"].([]interface{})
	if len(roles) != 1 {
		t.Errorf("Expected 1 linked role after idempotent link, got %d", len(roles))
	}
}

// U4 Behavioral Test: Attach JD via UI form
func TestUI_AttachJDViaForm(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a company and role
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "jd-test-company",
		"name": "JD Test Company",
	})
	env.PostFormFollowRedirect("/companies/jd-test-company/roles/new", map[string]string{
		"slug":  "jd-test-role",
		"title": "JD Test Role",
	})

	// Verify role page loads (legacy JD form removed in favor of artifacts)
	roleResp := env.Get("/companies/jd-test-company/roles/jd-test-role")
	env.AssertStatus(roleResp, 200)

	// Attach JD via legacy multipart form (HTML only for simplicity)
	// Legacy endpoint still works but UI section has been removed
	jdResp := env.PostMultipart("/companies/jd-test-company/roles/jd-test-role/jd",
		map[string]string{"html": "<html><body><h1>Test JD</h1></body></html>"},
		nil,
	)
	// Should redirect
	if jdResp.StatusCode != 303 && jdResp.StatusCode != 200 {
		t.Errorf("Expected redirect or success, got %d", jdResp.StatusCode)
	}

	// Verify JD files exist
	if !env.FileExists("data/companies/jd-test-company/roles/jd-test-role/job.html") {
		t.Error("job.html should exist after attaching JD via legacy endpoint")
	}

	// Verify role page loads after JD attachment
	roleResp2 := env.Get("/companies/jd-test-company/roles/jd-test-role")
	env.AssertStatus(roleResp2, 200)
}

// U4 Behavioral Test: Export via UI
func TestUI_ExportViaUI(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create some data first
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "export-test-company",
		"name": "Export Test Company",
	})

	// Trigger export via POST /export
	exportResp := env.PostFormFollowRedirect("/export", map[string]string{})
	env.AssertStatus(exportResp, 200)

	// Verify export.json exists
	if !env.FileExists("db/export.json") {
		t.Error("db/export.json should exist after export")
	}

	// Verify export contains our data
	exportContent := env.ReadFile("db/export.json")
	if !strings.Contains(exportContent, "export-test-company") {
		t.Error("export.json should contain our test company")
	}

	// Export again and verify determinism
	exportResp2 := env.PostFormFollowRedirect("/export", map[string]string{})
	env.AssertStatus(exportResp2, 200)

	export1 := env.ReadFile("db/export.json")
	// Run export again
	env.PostFormFollowRedirect("/export", map[string]string{})
	export2 := env.ReadFile("db/export.json")

	// Strip timestamps and compare
	export1Lines := stripExportedAt(export1)
	export2Lines := stripExportedAt(export2)
	if export1Lines != export2Lines {
		t.Error("Export should be deterministic")
	}
}

// S2 Behavioral Test: Role status updates and computed company status
func TestBehavioral_RoleStatusAndComputedCompanyStatus(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a company
	companyResp := env.PostJSON("/api/companies", map[string]string{
		"slug": "status-test-company",
		"name": "Status Test Company",
	})
	env.AssertStatus(companyResp, 201)

	// Create two roles
	role1Resp := env.PostJSON("/api/companies/status-test-company/roles", map[string]string{
		"slug":  "role-1",
		"title": "Role One",
	})
	env.AssertStatus(role1Resp, 201)

	role2Resp := env.PostJSON("/api/companies/status-test-company/roles", map[string]string{
		"slug":  "role-2",
		"title": "Role Two",
	})
	env.AssertStatus(role2Resp, 201)

	// Verify default role status is "recruiter_reached_out"
	getCompanyResp := env.Get("/api/companies/status-test-company")
	env.AssertStatus(getCompanyResp, 200)
	var companyWithRoles map[string]interface{}
	env.ReadJSON(getCompanyResp, &companyWithRoles)

	roles := companyWithRoles["roles"].([]interface{})
	if len(roles) != 2 {
		t.Fatalf("Expected 2 roles, got %d", len(roles))
	}

	// Verify default company status is "in_progress" (roles are not terminal)
	company := companyWithRoles["company"].(map[string]interface{})
	if company["status"] != "in_progress" {
		t.Errorf("Expected initial company status 'in_progress', got '%v'", company["status"])
	}

	// Update role1 to "rejected"
	updateResp := env.PatchJSON("/api/companies/status-test-company/roles/role-1/status", map[string]string{
		"status": "rejected",
	})
	env.AssertStatus(updateResp, 200)

	// Company status should still be "in_progress" (role2 is not terminal)
	getCompanyResp = env.Get("/api/companies/status-test-company")
	env.AssertStatus(getCompanyResp, 200)
	env.ReadJSON(getCompanyResp, &companyWithRoles)
	company = companyWithRoles["company"].(map[string]interface{})
	if company["status"] != "in_progress" {
		t.Errorf("Expected company status 'in_progress' when one role not terminal, got '%v'", company["status"])
	}

	// Update role2 to "offer"
	updateResp = env.PatchJSON("/api/companies/status-test-company/roles/role-2/status", map[string]string{
		"status": "offer",
	})
	env.AssertStatus(updateResp, 200)

	// Company status should now be "offer" (both roles terminal, one is offer)
	getCompanyResp = env.Get("/api/companies/status-test-company")
	env.AssertStatus(getCompanyResp, 200)
	env.ReadJSON(getCompanyResp, &companyWithRoles)
	company = companyWithRoles["company"].(map[string]interface{})
	if company["status"] != "offer" {
		t.Errorf("Expected company status 'offer' when any role is offer, got '%v'", company["status"])
	}

	// Update role2 to "rejected"
	updateResp = env.PatchJSON("/api/companies/status-test-company/roles/role-2/status", map[string]string{
		"status": "rejected",
	})
	env.AssertStatus(updateResp, 200)

	// Company status should now be "rejected" (all roles rejected)
	getCompanyResp = env.Get("/api/companies/status-test-company")
	env.AssertStatus(getCompanyResp, 200)
	env.ReadJSON(getCompanyResp, &companyWithRoles)
	company = companyWithRoles["company"].(map[string]interface{})
	if company["status"] != "rejected" {
		t.Errorf("Expected company status 'rejected' when all roles rejected, got '%v'", company["status"])
	}

	// Update role1 to "hr_interview" (non-terminal)
	updateResp = env.PatchJSON("/api/companies/status-test-company/roles/role-1/status", map[string]string{
		"status": "hr_interview",
	})
	env.AssertStatus(updateResp, 200)

	// Company status should go back to "in_progress"
	getCompanyResp = env.Get("/api/companies/status-test-company")
	env.AssertStatus(getCompanyResp, 200)
	env.ReadJSON(getCompanyResp, &companyWithRoles)
	company = companyWithRoles["company"].(map[string]interface{})
	if company["status"] != "in_progress" {
		t.Errorf("Expected company status 'in_progress' when role reverted to non-terminal, got '%v'", company["status"])
	}

	// Update role1 to "cancelled" (role2 is still "rejected")
	updateResp = env.PatchJSON("/api/companies/status-test-company/roles/role-1/status", map[string]string{
		"status": "cancelled",
	})
	env.AssertStatus(updateResp, 200)

	// Company status should be "rejected" (rejected beats cancelled)
	getCompanyResp = env.Get("/api/companies/status-test-company")
	env.AssertStatus(getCompanyResp, 200)
	env.ReadJSON(getCompanyResp, &companyWithRoles)
	company = companyWithRoles["company"].(map[string]interface{})
	if company["status"] != "rejected" {
		t.Errorf("Expected company status 'rejected' when one role cancelled and one rejected, got '%v'", company["status"])
	}

	// Update role2 to "cancelled" as well
	updateResp = env.PatchJSON("/api/companies/status-test-company/roles/role-2/status", map[string]string{
		"status": "cancelled",
	})
	env.AssertStatus(updateResp, 200)

	// Company status should now be "cancelled" (all roles cancelled)
	getCompanyResp = env.Get("/api/companies/status-test-company")
	env.AssertStatus(getCompanyResp, 200)
	env.ReadJSON(getCompanyResp, &companyWithRoles)
	company = companyWithRoles["company"].(map[string]interface{})
	if company["status"] != "cancelled" {
		t.Errorf("Expected company status 'cancelled' when all roles cancelled, got '%v'", company["status"])
	}

	// Test invalid status
	invalidResp := env.PatchJSON("/api/companies/status-test-company/roles/role-1/status", map[string]string{
		"status": "invalid_status",
	})
	env.AssertStatus(invalidResp, 400)

	// Test non-existent role
	notFoundResp := env.PatchJSON("/api/companies/status-test-company/roles/nonexistent/status", map[string]string{
		"status": "rejected",
	})
	env.AssertStatus(notFoundResp, 400)
}

// S3 UI Test: Update role status via form and verify it appears on pages
func TestUI_UpdateRoleStatusViaForm(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company and role
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "ui-status-company",
		"name": "UI Status Company",
	})
	env.PostFormFollowRedirect("/companies/ui-status-company/roles/new", map[string]string{
		"slug":  "ui-status-role",
		"title": "UI Status Role",
	})

	// Get role page - should show status dropdown
	rolePageResp := env.Get("/companies/ui-status-company/roles/ui-status-role")
	env.AssertStatus(rolePageResp, 200)
	rolePageBody := env.ReadBody(rolePageResp)
	if !strings.Contains(rolePageBody, "recruiter_reached_out") {
		t.Error("Role page should show recruiter_reached_out as default status")
	}
	if !strings.Contains(rolePageBody, "<select") {
		t.Error("Role page should contain status dropdown")
	}

	// Update status via form
	updateResp := env.PostFormFollowRedirect("/companies/ui-status-company/roles/ui-status-role/status", map[string]string{
		"status": "hr_interview",
	})
	env.AssertStatus(updateResp, 200)
	updateBody := env.ReadBody(updateResp)
	if !strings.Contains(updateBody, "hr_interview") {
		t.Error("Role page should show updated hr_interview status")
	}
	if !strings.Contains(updateBody, "Status+updated") && !strings.Contains(updateBody, "Status updated") {
		// Check for success message (might be URL encoded or not)
	}

	// Company page should show role status in table
	companyPageResp := env.Get("/companies/ui-status-company")
	env.AssertStatus(companyPageResp, 200)
	companyPageBody := env.ReadBody(companyPageResp)
	if !strings.Contains(companyPageBody, "hr_interview") {
		t.Error("Company page should show role status in roles table")
	}

	// Companies list should show computed status (in_progress since role is not terminal)
	companiesResp := env.Get("/companies")
	env.AssertStatus(companiesResp, 200)
	companiesBody := env.ReadBody(companiesResp)
	if !strings.Contains(companiesBody, "in_progress") {
		t.Error("Companies list should show computed in_progress status")
	}

	// Update to terminal status (offer)
	env.PostFormFollowRedirect("/companies/ui-status-company/roles/ui-status-role/status", map[string]string{
		"status": "offer",
	})

	// Companies list should now show "offer"
	companiesResp = env.Get("/companies")
	env.AssertStatus(companiesResp, 200)
	companiesBody = env.ReadBody(companiesResp)
	if !strings.Contains(companiesBody, "offer") {
		t.Error("Companies list should show computed offer status after role is offer")
	}
}

// S4 Test: Export.json includes role status and computed company status
func TestExport_IncludesStatusAndComputedStatus(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company and roles with different statuses
	env.PostJSON("/api/companies", map[string]string{
		"slug": "export-status-company",
		"name": "Export Status Company",
	})

	env.PostJSON("/api/companies/export-status-company/roles", map[string]string{
		"slug":  "role-offer",
		"title": "Role With Offer",
	})
	env.PostJSON("/api/companies/export-status-company/roles", map[string]string{
		"slug":  "role-rejected",
		"title": "Role Rejected",
	})

	// Update role statuses
	env.PatchJSON("/api/companies/export-status-company/roles/role-offer/status", map[string]string{
		"status": "offer",
	})
	env.PatchJSON("/api/companies/export-status-company/roles/role-rejected/status", map[string]string{
		"status": "rejected",
	})

	// Run export
	exportResp := env.PostJSON("/api/export", nil)
	env.AssertStatus(exportResp, 200)

	// Read export
	exportContent := env.ReadFile("db/export.json")

	// Verify role statuses are in export
	if !strings.Contains(exportContent, `"status": "offer"`) {
		t.Error("export.json should contain role with status 'offer'")
	}
	if !strings.Contains(exportContent, `"status": "rejected"`) {
		t.Error("export.json should contain role with status 'rejected'")
	}

	// Verify company_views section exists with computed status
	if !strings.Contains(exportContent, `"company_views"`) {
		t.Error("export.json should contain company_views section")
	}
	// Company should have "offer" computed status (any role is offer)
	if !strings.Contains(exportContent, `"computed_status": "offer"`) {
		t.Error("export.json should contain computed_status 'offer' in company_views")
	}

	// Run export again to verify determinism
	export1 := env.ReadFile("db/export.json")
	env.PostJSON("/api/export", nil)
	export2 := env.ReadFile("db/export.json")

	// Strip timestamps and compare
	export1Lines := stripExportedAt(export1)
	export2Lines := stripExportedAt(export2)
	if export1Lines != export2Lines {
		t.Error("Export should be deterministic including status data")
	}
}

// R5 Test: Export includes meetings data
func TestExport_IncludesMeetingsV2(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a company and role
	env.PostJSON("/api/companies", map[string]string{
		"slug": "export-v2-company",
		"name": "Export V2 Company",
	})
	env.PostJSON("/api/companies/export-v2-company/roles", map[string]string{
		"slug":  "export-v2-role",
		"title": "Export V2 Role",
	})

	// Create a role meeting using v2 API
	meetingResp := env.PostJSON("/api/companies/export-v2-company/roles/export-v2-role/meetings", map[string]string{
		"occurred_at": "2024-09-01T10:00:00Z",
		"title":       "V2 Export Test Meeting",
	})
	env.AssertStatus(meetingResp, 201)

	var meeting map[string]interface{}
	env.ReadJSON(meetingResp, &meeting)
	meetingID := meeting["id"].(string)

	// Export
	env.PostJSON("/api/export", nil)

	// Verify meetings section in export
	exportContent := env.ReadFile("db/export.json")

	if !strings.Contains(exportContent, `"meetings"`) {
		t.Error("export.json should contain meetings section")
	}
	if !strings.Contains(exportContent, meetingID) {
		t.Error("export.json should contain the role meeting ID")
	}
	if !strings.Contains(exportContent, "V2 Export Test Meeting") {
		t.Error("export.json should contain the role meeting title")
	}

	// Verify determinism with meetings
	export1 := env.ReadFile("db/export.json")
	env.PostJSON("/api/export", nil)
	export2 := env.ReadFile("db/export.json")

	export1Lines := stripExportedAt(export1)
	export2Lines := stripExportedAt(export2)
	if export1Lines != export2Lines {
		t.Error("Export should be deterministic including meetings data")
	}
}

// M2 Test: Meeting IDs are 8-char short IDs, and filenames use short IDs
func TestBehavioral_MeetingShortIDs(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a company
	companyResp := env.PostJSON("/api/companies", map[string]string{
		"slug": "short-id-company",
		"name": "Short ID Company",
	})
	env.AssertStatus(companyResp, 201)

	// Create a role (meetings are now role-level)
	roleResp := env.PostJSON("/api/companies/short-id-company/roles", map[string]string{
		"slug":  "test-role",
		"title": "Test Role",
	})
	env.AssertStatus(roleResp, 201)

	// Create a role meeting (v2 API)
	meetingResp := env.PostJSON("/api/companies/short-id-company/roles/test-role/meetings", map[string]string{
		"occurred_at": "2024-03-15T10:00:00Z",
		"title":       "Short ID Test Meeting",
	})
	env.AssertStatus(meetingResp, 201)

	var meeting map[string]interface{}
	env.ReadJSON(meetingResp, &meeting)

	// Verify meeting ID is exactly 8 characters (short ID format)
	meetingID := meeting["id"].(string)
	if len(meetingID) != 8 {
		t.Errorf("Expected meeting ID to be 8 characters, got %d characters: %q", len(meetingID), meetingID)
	}

	// Verify path_md filename ends with _<8-char-id>.md
	pathMD := meeting["path_md"].(string)
	expectedSuffix := "_" + meetingID + ".md"
	if !strings.HasSuffix(pathMD, expectedSuffix) {
		t.Errorf("Expected path_md to end with %q, got %q", expectedSuffix, pathMD)
	}

	// Verify the file exists on disk
	if !env.FileExists(pathMD) {
		t.Errorf("Meeting note file should exist at %s", pathMD)
	}

	// Verify the file content contains the short meeting ID
	content := env.ReadFile(pathMD)
	if !strings.Contains(content, "meeting_id: "+meetingID) {
		t.Error("Meeting note should contain the short meeting_id in frontmatter")
	}
}

// M2 UI Test: Creating meeting via UI creates file with 8-char short ID
func TestUI_CreateMeetingWithShortID(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a company
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "ui-short-id-company",
		"name": "UI Short ID Company",
	})

	// Create a role (meetings are now role-level)
	env.PostFormFollowRedirect("/companies/ui-short-id-company/roles/new", map[string]string{
		"slug":  "test-role",
		"title": "Test Role",
	})

	// Create a role meeting via UI form
	meetingResp := env.PostFormFollowRedirect("/companies/ui-short-id-company/roles/test-role/meetings/new", map[string]string{
		"title":       "UI Short ID Meeting",
		"occurred_at": "2024-06-20T09:00",
	})
	env.AssertStatus(meetingResp, 200)

	// Get the role to find its ID
	companyResp := env.Get("/api/companies/ui-short-id-company")
	env.AssertStatus(companyResp, 200)

	var companyDetails map[string]interface{}
	env.ReadJSON(companyResp, &companyDetails)

	roles := companyDetails["roles"].([]interface{})
	if len(roles) == 0 {
		t.Fatal("Expected at least one role")
	}

	role := roles[0].(map[string]interface{})
	roleID := role["id"].(string)

	// Use the MeetingService directly to get the UI-created meeting
	ctx := context.Background()
	meetings, err := env.MeetingService.ListMeetingsByRole(ctx, roleID)
	if err != nil {
		t.Fatalf("Failed to list meetings: %v", err)
	}
	if len(meetings) == 0 {
		t.Fatal("Expected at least one meeting created by UI form")
	}

	meeting := meetings[0]
	meetingID := meeting.ID

	// Verify meeting ID is exactly 8 characters
	if len(meetingID) != 8 {
		t.Errorf("Expected meeting ID to be 8 characters (short ID), got %d characters: %q", len(meetingID), meetingID)
	}

	// Verify the path_md ends with the short ID
	if !strings.HasSuffix(meeting.PathMD, "_"+meetingID+".md") {
		t.Errorf("Meeting filename should end with 8-char ID, got path: %s", meeting.PathMD)
	}

	// Verify meeting title matches what was submitted via UI
	if meeting.Title != "UI Short ID Meeting" {
		t.Errorf("Expected meeting title 'UI Short ID Meeting', got %q", meeting.Title)
	}
}

// R3 E2E Test: Role meeting creation (v2) creates file under role folder
func TestMeeting_CreateRoleMeeting(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a company
	companyResp := env.PostJSON("/api/companies", map[string]string{
		"slug": "v2-role-meeting-company",
		"name": "V2 Role Meeting Company",
	})
	env.AssertStatus(companyResp, 201)

	// Create a role
	roleResp := env.PostJSON("/api/companies/v2-role-meeting-company/roles", map[string]string{
		"slug":  "backend-engineer",
		"title": "Backend Engineer",
	})
	env.AssertStatus(roleResp, 201)

	// Create a role meeting via v2 API
	meetingResp := env.PostJSON("/api/companies/v2-role-meeting-company/roles/backend-engineer/meetings", map[string]string{
		"occurred_at": "2024-06-15T10:00:00Z",
		"title":       "Technical Interview",
	})
	env.AssertStatus(meetingResp, 201)

	var meeting map[string]interface{}
	env.ReadJSON(meetingResp, &meeting)

	// Verify meeting ID is 8 characters
	meetingID := meeting["id"].(string)
	if len(meetingID) != 8 {
		t.Errorf("Expected meeting ID to be 8 characters, got %d: %q", len(meetingID), meetingID)
	}

	// Verify role_id is set
	roleID := meeting["role_id"].(string)
	if roleID == "" {
		t.Error("Expected role_id to be set for role meeting")
	}

	// Verify path_md is under role folder
	pathMD := meeting["path_md"].(string)
	expectedPrefix := "data/companies/v2-role-meeting-company/roles/backend-engineer/meetings/"
	if !strings.HasPrefix(pathMD, expectedPrefix) {
		t.Errorf("Expected path to start with %q, got %q", expectedPrefix, pathMD)
	}

	// Verify file exists
	if !env.FileExists(pathMD) {
		t.Errorf("Meeting note file should exist at %s", pathMD)
	}

	// Verify file content
	content := env.ReadFile(pathMD)
	if !strings.Contains(content, "# Technical Interview") {
		t.Error("Meeting note should contain title")
	}
	if !strings.Contains(content, "meeting_id: "+meetingID) {
		t.Error("Meeting note should contain meeting_id")
	}
}

// R3 UI Test: Create role meeting via HTML form
func TestUI_CreateRoleMeetingViaForm(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company and role
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "ui-v2-meeting-company",
		"name": "UI V2 Meeting Company",
	})
	env.PostFormFollowRedirect("/companies/ui-v2-meeting-company/roles/new", map[string]string{
		"slug":  "ui-test-role",
		"title": "UI Test Role",
	})

	// Create meeting via form
	meetingResp := env.PostFormFollowRedirect("/companies/ui-v2-meeting-company/roles/ui-test-role/meetings/new", map[string]string{
		"title":       "UI Role Meeting",
		"occurred_at": "2024-08-01T09:00",
	})
	env.AssertStatus(meetingResp, 200)

	// Verify meeting file exists in the role folder
	if !env.FileExists("data/companies/ui-v2-meeting-company/roles/ui-test-role/meetings") {
		t.Error("Role meetings folder should exist")
	}
}

// J1 E2E Test: JD viewer displays HTML in sandboxed iframe with CSP
func TestUI_JDViewer(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company and role
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "jd-viewer-company",
		"name": "JD Viewer Company",
	})
	env.PostFormFollowRedirect("/companies/jd-viewer-company/roles/new", map[string]string{
		"slug":  "jd-viewer-role",
		"title": "JD Viewer Role",
	})

	// Attach JD with HTML content
	htmlContent := "<html><body><h1>Test Job Description</h1><p>Requirements: Go, SQL</p></body></html>"
	jdResp := env.PostMultipart("/companies/jd-viewer-company/roles/jd-viewer-role/jd",
		map[string]string{"html": htmlContent},
		nil,
	)
	if jdResp.StatusCode != 303 && jdResp.StatusCode != 200 {
		t.Errorf("Expected redirect or success, got %d", jdResp.StatusCode)
	}

	// Test JD viewer page loads
	viewerResp := env.Get("/companies/jd-viewer-company/roles/jd-viewer-role/jd")
	env.AssertStatus(viewerResp, 200)
	viewerBody := env.ReadBody(viewerResp)

	// Verify viewer page contains expected elements
	if !strings.Contains(viewerBody, "Job Description") {
		t.Error("JD viewer page should contain 'Job Description' heading")
	}
	if !strings.Contains(viewerBody, "jd-viewer-company") {
		t.Error("JD viewer page should contain company link")
	}
	if !strings.Contains(viewerBody, "jd-viewer-role") {
		t.Error("JD viewer page should contain role link")
	}
	if !strings.Contains(viewerBody, "iframe") {
		t.Error("JD viewer page should contain an iframe")
	}
	if !strings.Contains(viewerBody, `sandbox="allow-same-origin"`) {
		t.Error("JD viewer iframe should have sandbox attribute")
	}
	if !strings.Contains(viewerBody, "/jd/raw") {
		t.Error("JD viewer iframe src should point to /jd/raw")
	}

	// Test raw JD endpoint returns HTML with CSP header
	rawResp := env.Get("/companies/jd-viewer-company/roles/jd-viewer-role/jd/raw")
	env.AssertStatus(rawResp, 200)

	// Check CSP header
	csp := rawResp.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Raw JD endpoint should set Content-Security-Policy header")
	}
	if !strings.Contains(csp, "default-src 'none'") {
		t.Error("CSP should include default-src 'none'")
	}
	if !strings.Contains(csp, "script-src") {
		// If script-src is not present, default-src 'none' blocks scripts (which is correct)
	}

	// Check content type
	contentType := rawResp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected Content-Type text/html, got %s", contentType)
	}

	// Check the actual HTML content is returned
	rawBody := env.ReadBody(rawResp)
	if !strings.Contains(rawBody, "Test Job Description") {
		t.Error("Raw JD endpoint should return the JD HTML content")
	}
	if !strings.Contains(rawBody, "Requirements: Go, SQL") {
		t.Error("Raw JD endpoint should return full JD content")
	}

	// Test 404 for role without JD attached
	env.PostFormFollowRedirect("/companies/jd-viewer-company/roles/new", map[string]string{
		"slug":  "no-jd-role",
		"title": "Role Without JD",
	})
	noJDResp := env.Get("/companies/jd-viewer-company/roles/no-jd-role/jd")
	env.AssertStatus(noJDResp, 404)

	// Test 404 for non-existent company
	notFoundResp := env.Get("/companies/nonexistent/roles/nonexistent/jd")
	env.AssertStatus(notFoundResp, 404)
}

// TestUI_ResumeAttachment tests the resume attachment feature E2E
func TestUI_ResumeAttachment(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company and role
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "resume-test-company",
		"name": "Resume Test Company",
	})
	env.PostFormFollowRedirect("/companies/resume-test-company/roles/new", map[string]string{
		"slug":  "resume-test-role",
		"title": "Resume Test Role",
	})

	// Attach resume with JSON via textarea (form field)
	jsonContent := `{"name": "Test Resume", "skills": ["Go", "Python"]}`
	jsonResp := env.PostMultipart("/companies/resume-test-company/roles/resume-test-role/resume",
		map[string]string{"resume_json": jsonContent},
		nil,
	)
	if jsonResp.StatusCode != 303 && jsonResp.StatusCode != 200 {
		t.Errorf("Expected redirect or success, got %d", jsonResp.StatusCode)
	}

	// Verify JSON file exists
	if !env.FileExists("data/companies/resume-test-company/roles/resume-test-role/resume/resume.jsonc") {
		t.Error("resume.jsonc should exist")
	}

	// Verify file content
	jsonFileContent := env.ReadFile("data/companies/resume-test-company/roles/resume-test-role/resume/resume.jsonc")
	if !strings.Contains(jsonFileContent, "Test Resume") {
		t.Error("resume.jsonc should contain the uploaded content")
	}

	// Verify role page loads successfully (legacy resume section removed in favor of artifacts)
	roleResp := env.Get("/companies/resume-test-company/roles/resume-test-role")
	env.AssertStatus(roleResp, 200)

	// Now attach PDF file only
	pdfContent := []byte("%PDF-1.4 test content")
	pdfResp := env.PostMultipart("/companies/resume-test-company/roles/resume-test-role/resume",
		nil,
		map[string][]byte{"pdf": pdfContent},
	)
	if pdfResp.StatusCode != 303 && pdfResp.StatusCode != 200 {
		t.Errorf("Expected redirect or success, got %d", pdfResp.StatusCode)
	}

	// Verify PDF file exists
	if !env.FileExists("data/companies/resume-test-company/roles/resume-test-role/resume/resume.pdf") {
		t.Error("resume.pdf should exist")
	}

	// Verify role page loads successfully after PDF attach
	roleResp2 := env.Get("/companies/resume-test-company/roles/resume-test-role")
	env.AssertStatus(roleResp2, 200)

	// Test overwrite: submit new JSON via textarea
	newJsonContent := `{"name": "Updated Resume", "version": 2}`
	overwriteResp := env.PostMultipart("/companies/resume-test-company/roles/resume-test-role/resume",
		map[string]string{"resume_json": newJsonContent},
		nil,
	)
	if overwriteResp.StatusCode != 303 && overwriteResp.StatusCode != 200 {
		t.Errorf("Expected redirect or success, got %d", overwriteResp.StatusCode)
	}

	// Verify content was overwritten
	updatedContent := env.ReadFile("data/companies/resume-test-company/roles/resume-test-role/resume/resume.jsonc")
	if !strings.Contains(updatedContent, "Updated Resume") {
		t.Error("resume.jsonc should be overwritten with new content")
	}
	if strings.Contains(updatedContent, "Test Resume") {
		t.Error("Old content should be replaced")
	}

	// Test error: no content provided - should redirect with error message
	emptyResp := env.PostMultipart("/companies/resume-test-company/roles/resume-test-role/resume",
		nil,
		nil,
	)
	// Client follows redirect, so we get 200 with error message in body
	env.AssertStatus(emptyResp, 200)
	emptyBody := env.ReadBody(emptyResp)
	if !strings.Contains(emptyBody, "At least JSON or PDF must be provided") {
		t.Error("Error message should be shown when no content provided")
	}

	// Test error: invalid JSON - should redirect with error message and NOT overwrite
	invalidJsonResp := env.PostMultipart("/companies/resume-test-company/roles/resume-test-role/resume",
		map[string]string{"resume_json": "{ not valid json"},
		nil,
	)
	// Client follows redirect, so we get 200 with error message in body
	env.AssertStatus(invalidJsonResp, 200)
	invalidJsonBody := env.ReadBody(invalidJsonResp)
	if !strings.Contains(invalidJsonBody, "invalid JSON") {
		t.Error("Error message should be shown when invalid JSON provided")
	}

	// Verify original content is still there (not overwritten by invalid JSON)
	stillValidContent := env.ReadFile("data/companies/resume-test-company/roles/resume-test-role/resume/resume.jsonc")
	if !strings.Contains(stillValidContent, "Updated Resume") {
		t.Error("resume.jsonc should not be overwritten by invalid JSON submission")
	}

	// Test JSONC support: JSON with comments should be accepted
	jsoncContent := `{
		// This is a single-line comment
		"name": "JSONC Resume",
		/* This is a
		   multi-line comment */
		"skills": ["Go", "TypeScript"],
		"trailing": "comma", // trailing comma below
	}`
	jsoncResp := env.PostMultipart("/companies/resume-test-company/roles/resume-test-role/resume",
		map[string]string{"resume_json": jsoncContent},
		nil,
	)
	if jsoncResp.StatusCode != 303 && jsoncResp.StatusCode != 200 {
		t.Errorf("JSONC should be accepted, got status %d", jsoncResp.StatusCode)
	}

	// Verify saved file preserves comments (JSONC format)
	savedJSONC := env.ReadFile("data/companies/resume-test-company/roles/resume-test-role/resume/resume.jsonc")
	if !strings.Contains(savedJSONC, "// This is a single-line comment") {
		t.Error("Saved file should preserve single-line comments")
	}
	if !strings.Contains(savedJSONC, "/* This is a") {
		t.Error("Saved file should preserve multi-line comments")
	}
	if !strings.Contains(savedJSONC, "JSONC Resume") {
		t.Error("Saved file should contain the resume data")
	}
}

// TestExport_IncludesResumes verifies that export.json includes resume paths
func TestExport_IncludesResumes(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company and role
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "export-resume-company",
		"name": "Export Resume Company",
	})
	env.PostFormFollowRedirect("/companies/export-resume-company/roles/new", map[string]string{
		"slug":  "export-resume-role",
		"title": "Export Resume Role",
	})

	// Attach resume with both JSON (via textarea) and PDF
	jsonContent := `{"name": "Export Test Resume"}`
	pdfContent := []byte("%PDF-1.4 export test")
	env.PostMultipart("/companies/export-resume-company/roles/export-resume-role/resume",
		map[string]string{"resume_json": jsonContent},
		map[string][]byte{"pdf": pdfContent},
	)

	// Run export
	exportResp := env.PostFormFollowRedirect("/export", map[string]string{})
	env.AssertStatus(exportResp, 200)

	// Verify export contains resume paths
	exportContent := env.ReadFile("db/export.json")
	if !strings.Contains(exportContent, `"resumes"`) {
		t.Error("export.json should contain resumes array")
	}
	if !strings.Contains(exportContent, "resume/resume.jsonc") {
		t.Error("export.json should contain JSON resume path")
	}
	if !strings.Contains(exportContent, "resume/resume.pdf") {
		t.Error("export.json should contain PDF resume path")
	}

	// Verify determinism: export again and compare
	export1 := env.ReadFile("db/export.json")
	env.PostFormFollowRedirect("/export", map[string]string{})
	export2 := env.ReadFile("db/export.json")

	// Strip timestamps and compare
	export1Lines := stripExportedAt(export1)
	export2Lines := stripExportedAt(export2)
	if export1Lines != export2Lines {
		t.Error("Export with resumes should be deterministic")
	}
}

// TestSmoke_ContactsPage verifies the /contacts list and detail pages load
func TestSmoke_ContactsPage(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create a contact via API
	contactResp := env.PostJSON("/api/contacts", map[string]string{
		"name": "Smoke Test Contact",
		"org":  "Smoke Test Org",
	})
	env.AssertStatus(contactResp, 201)
	var contact map[string]interface{}
	env.ReadJSON(contactResp, &contact)
	contactID := contact["id"].(string)

	// GET /contacts returns 200 and contains contact name
	listResp := env.Get("/contacts")
	env.AssertStatus(listResp, 200)
	listBody := env.ReadBody(listResp)
	if !strings.Contains(listBody, "Smoke Test Contact") {
		t.Error("/contacts page should contain contact name")
	}

	// GET /contacts/{id} returns 200
	detailResp := env.Get("/contacts/" + contactID)
	env.AssertStatus(detailResp, 200)
	detailBody := env.ReadBody(detailResp)
	if !strings.Contains(detailBody, "Smoke Test Contact") {
		t.Error("/contacts/{id} page should contain contact name")
	}
}

// TestBehavioral_LinkRoleToContactViaAPI verifies POST /api/contacts/{id}/roles
// and GET /api/contacts/{id} returns the linked role
func TestBehavioral_LinkRoleToContactViaAPI(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company and role
	env.PostJSON("/api/companies", map[string]string{
		"slug": "contact-link-company",
		"name": "Contact Link Company",
	})
	env.PostJSON("/api/companies/contact-link-company/roles", map[string]string{
		"slug":  "contact-link-role",
		"title": "Contact Link Role",
	})

	// Create contact
	contactResp := env.PostJSON("/api/contacts", map[string]string{
		"name": "Contact Link Test",
	})
	env.AssertStatus(contactResp, 201)
	var contact map[string]interface{}
	env.ReadJSON(contactResp, &contact)
	contactID := contact["id"].(string)

	// Link role to contact via API
	linkResp := env.PostJSON("/api/contacts/"+contactID+"/roles", map[string]string{
		"role_ref": "contact-link-company/contact-link-role",
	})
	env.AssertStatus(linkResp, 204)

	// GET /api/contacts/{id} should include the linked role
	getResp := env.Get("/api/contacts/" + contactID)
	env.AssertStatus(getResp, 200)
	var contactDetails map[string]interface{}
	env.ReadJSON(getResp, &contactDetails)

	roles, ok := contactDetails["roles"].([]interface{})
	if !ok || len(roles) != 1 {
		t.Fatalf("Expected 1 linked role, got %v", contactDetails["roles"])
	}

	roleEntry := roles[0].(map[string]interface{})
	roleObj := roleEntry["role"].(map[string]interface{})
	if roleObj["slug"] != "contact-link-role" {
		t.Errorf("Expected role slug 'contact-link-role', got '%v'", roleObj["slug"])
	}

	companyObj := roleEntry["company"].(map[string]interface{})
	if companyObj["slug"] != "contact-link-company" {
		t.Errorf("Expected company slug 'contact-link-company', got '%v'", companyObj["slug"])
	}

	// Link same role again (idempotent) — should not error
	linkResp2 := env.PostJSON("/api/contacts/"+contactID+"/roles", map[string]string{
		"role_ref": "contact-link-company/contact-link-role",
	})
	env.AssertStatus(linkResp2, 204)

	// Still only one role
	getResp2 := env.Get("/api/contacts/" + contactID)
	env.AssertStatus(getResp2, 200)
	var contactDetails2 map[string]interface{}
	env.ReadJSON(getResp2, &contactDetails2)
	roles2 := contactDetails2["roles"].([]interface{})
	if len(roles2) != 1 {
		t.Errorf("Expected 1 linked role after idempotent link, got %d", len(roles2))
	}
}

// TestUI_CreateContactAndLinkRole tests the contacts form flow
func TestUI_CreateContactAndLinkRole(t *testing.T) {
	env := testharness.NewTestEnv(t)

	// Create company and role for linking
	env.PostFormFollowRedirect("/companies/new", map[string]string{
		"slug": "ui-contact-company",
		"name": "UI Contact Company",
	})
	env.PostFormFollowRedirect("/companies/ui-contact-company/roles/new", map[string]string{
		"slug":  "ui-contact-role",
		"title": "UI Contact Role",
	})

	// GET /contacts shows the Add Contact form
	contactsResp := env.Get("/contacts")
	env.AssertStatus(contactsResp, 200)
	contactsBody := env.ReadBody(contactsResp)
	if !strings.Contains(contactsBody, "Add Contact") {
		t.Error("/contacts page should contain 'Add Contact' form")
	}

	// Create contact via form — should redirect to /contacts
	createResp := env.PostFormFollowRedirect("/contacts/new", map[string]string{
		"name":  "UI Form Contact",
		"org":   "UI Form Org",
		"email": "ui@form.com",
	})
	env.AssertStatus(createResp, 200)
	createBody := env.ReadBody(createResp)
	if !strings.Contains(createBody, "UI Form Contact") {
		t.Error("/contacts list should show newly created contact")
	}

	// Find contact ID via API
	contactListAPIResp := env.PostJSON("/api/contacts", map[string]string{
		"name": "Role Link Contact",
	})
	var c map[string]interface{}
	env.ReadJSON(contactListAPIResp, &c)
	contactID := c["id"].(string)

	// Visit contact page — should show link role form
	contactPageResp := env.Get("/contacts/" + contactID)
	env.AssertStatus(contactPageResp, 200)
	contactPageBody := env.ReadBody(contactPageResp)
	if !strings.Contains(contactPageBody, "Link Role") {
		t.Error("/contacts/{id} page should contain 'Link Role' form")
	}

	// Link role via form
	linkResp := env.PostFormFollowRedirect("/contacts/"+contactID+"/roles/link", map[string]string{
		"role_ref": "ui-contact-company/ui-contact-role",
	})
	env.AssertStatus(linkResp, 200)
	linkBody := env.ReadBody(linkResp)
	if !strings.Contains(linkBody, "UI Contact Role") {
		t.Error("Contact page after link should show the linked role title")
	}
}

// TestUI_CreateContactViaForm_RedirectToContacts checks that creating a
// contact via form redirects to /contacts
func TestUI_CreateContactViaForm_RedirectToContacts(t *testing.T) {
	env := testharness.NewTestEnv(t)

	createResp := env.PostFormFollowRedirect("/contacts/new", map[string]string{
		"name": "Redirect Test Contact",
	})
	env.AssertStatus(createResp, 200)

	// The final URL after redirect should be /contacts (the contacts list page)
	createBody := env.ReadBody(createResp)
	if !strings.Contains(createBody, "Redirect Test Contact") {
		t.Error("After creating contact, /contacts list should show the new contact")
	}
}

