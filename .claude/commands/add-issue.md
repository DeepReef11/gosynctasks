---
description: Create GitHub issues and add them to ROADMAP.md without attempting to fix them
---

Use the Task tool to launch a general-purpose agent with the following instructions:

Your task is to create GitHub issues and add them to the project roadmap WITHOUT attempting to fix or implement them. Follow these steps:

1. **Create GitHub Issue**:
   - Use `gh issue create` to create the issue
   - Provide a clear, descriptive title
   - Include a detailed description in the issue body
   - Add appropriate labels if mentioned by the user
   - Capture the issue number from the output

2. **Determine Documentation Strategy**:
   - If the issue description is longer than 300 characters, create a separate documentation file
   - Documentation files should be placed in `docs/issues/issue-{NUMBER}.md`
   - Use the format: `docs/issues/issue-{NUMBER}.md` (e.g., `docs/issues/issue-123.md`)
   - Otherwise, include the full description directly in ROADMAP.md

3. **Update ROADMAP.md**:
   - Read the existing ROADMAP.md file
   - Add a new section or entry for this issue
   - Use this format for new issues:
     ```markdown
     ## Issue #{NUMBER}: {Title}

     **Status**: Open
     **Created**: {DATE}
     **Labels**: {LABELS if any}

     {DESCRIPTION or "See: docs/issues/issue-{NUMBER}.md for full details"}
     ```
   - If the issue relates to an existing roadmap section, add it there instead of creating a new section
   - Place new standalone issues at the end of the file before any "Future Enhancements" section

4. **Create Documentation File (if needed)**:
   - Create the `docs/issues/` directory if it doesn't exist
   - Write the full issue details to `docs/issues/issue-{NUMBER}.md`
   - Use this format:
     ```markdown
     # Issue #{NUMBER}: {Title}

     **GitHub Issue**: #{NUMBER}
     **Status**: Open
     **Created**: {DATE}
     **Labels**: {LABELS if any}

     ## Description

     {FULL_DESCRIPTION}

     ## Context

     {ANY_ADDITIONAL_CONTEXT}

     ## Acceptance Criteria

     {CRITERIA if provided}
     ```

5. **Return Summary**:
   - Provide the issue number and URL
   - Confirm whether it was added to ROADMAP.md inline or via separate doc
   - Show the file path if a separate doc was created

**IMPORTANT CONSTRAINTS**:
- DO NOT attempt to implement or fix the issue
- DO NOT create branches or make code changes
- ONLY create the issue, update documentation, and report back
- If the user asks you to also implement it, create the issue first, then ask if they want you to proceed with implementation

**Example execution flow**:
1. User provides issue details
2. Create GitHub issue → get #123
3. Description is 450 chars → create docs/issues/issue-123.md
4. Update ROADMAP.md with reference to docs/issues/issue-123.md
5. Report: "Created issue #123: {title}. Documentation: docs/issues/issue-123.md. Added to ROADMAP.md"
