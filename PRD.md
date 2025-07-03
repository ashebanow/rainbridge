# Instructions for creating a product requirements document (PRD)

You are a senior product manager and an expert in creating product requirements documents (PRDs) for software development teams.

Your task is to create a comprehensive product requirements document (PRD) for the following project:

<prd_instructions>

RainBridge is a simple utility to import bookmarks into Karakeep, using the Raindrop API. Key considerations are handling API rate limiting, secure authentication for the Raindrop.io API, and a user-friendly configuration for API keys and other settings. The goal is to provide a seamless migration path for users moving their bookmarks from Raindrop.io to Karakeep.

</prd_instructions>

Follow these steps to create the PRD:

<steps>

1. Begin with a brief overview explaining the project and the purpose of the document.

2. Use sentence case for all headings except for the title of the document, which can be title case, including any you create that are not included in the prd_outline below.

3. Under each main heading include relevant subheadings and fill them with details derived from the prd_instructions

4. Organize your PRD into the sections as shown in the prd_outline below

5. For each section of prd_outline, provide detailed and relevant information based on the PRD instructions. Ensure that you:
   - Use clear and concise language
   - Provide specific details and metrics where required
   - Maintain consistency throughout the document
   - Address all points mentioned in each section

6. When creating user stories and acceptance criteria:
	- List ALL necessary user stories including primary, alternative, and edge-case scenarios.
	- Assign a unique requirement ID (e.g., US-001) to each user story for direct traceability
	- Include at least one user story specifically for secure access or authentication if the application requires user identification or access restrictions
	- Ensure no potential user interaction is omitted
	- Make sure each user story is testable
	- Review the user_story example below for guidance on how to structure your user stories

7. After completing the PRD, review it against this Final Checklist:
   - Is each user story testable?
   - Are acceptance criteria clear and specific?
   - Do we have enough user stories to build a fully functional application for it?
   - Have we addressed authentication and authorization requirements (if applicable)?

8. Format your PRD:
   - Maintain consistent formatting and numbering.
  	- Do not use dividers or horizontal rules in the output.
  	- List ALL User Stories in the output!
	  - Format the PRD in valid Markdown, with no extraneous disclaimers.
	  - Do not add a conclusion or footer. The user_story section is the last section.
	  - Fix any grammatical errors in the prd_instructions and ensure proper casing of any names.
	  - When referring to the project, do not use project_title. Instead, refer to it in a more simple and conversational way. For example, "the project", "this tool" etc.

</steps>

<prd_outline>

# PRD: RainBridge

## 1. Product overview
### 1.1 Document title and version
   - PRD: RainBridge
   - Version: 0.1.0
### 1.2 Product summary
   RainBridge is a utility designed to streamline the process of importing bookmarks from Raindrop.io into Karakeep. It provides a simple and efficient way for users to migrate their curated content, ensuring that valuable links and associated metadata are transferred seamlessly.

   The tool will handle all API interactions with both services, transparently manage API rate limiting to ensure reliability, and provide a straightforward configuration process for the user. The primary goal is to lower the barrier to entry for users wishing to adopt Karakeep by removing the friction of manual data migration.

### 1.3 Distribution
   - The tool will be released as open-source software.
   - It will be distributed through popular package managers to ensure easy installation for users on different operating systems.
   - Target repositories include:
     - Homebrew (for macOS and Linux)
     - Arch User Repository (AUR)
     - Fedora (COPR or official repositories)
     - Debian/Ubuntu (PPA or official repositories)

## 2. Goals
### 2.1 Business goals
   - Increase adoption of the Karakeep platform by providing an easy migration path from Raindrop.io.
   - Enhance the Karakeep ecosystem with a valuable, community-oriented tool.
   - Position Karakeep as a user-friendly platform that is easy to switch to.
### 2.2 User goals
   - To import all my Raindrop.io bookmarks into Karakeep quickly and reliably.
   - To ensure that metadata such as titles, descriptions, and tags are preserved during the import.
   - To avoid manual, time-consuming data entry.
### 2.3 Non-goals
   - This tool will not be a full-featured client for either Raindrop.io or Karakeep.
   - It will not support ongoing, two-way synchronization between the two services.
   - It will not provide functionality for managing bookmarks within Raindrop.io (e.g., deleting, editing).
   - It will not have a graphical user interface (GUI) in its initial versions.

## 3. User personas
### 3.1 Key user types
   - Existing Karakeep users who want to consolidate their bookmarks from Raindrop.io.
   - Potential Karakeep users who are currently using Raindrop.io and are evaluating Karakeep.
   - Developers or technically-savvy users who are comfortable with command-line tools.
### 3.2 Basic persona details
   - **Alex, the Knowledge Hoarder**: Alex uses Raindrop.io extensively to save articles, tutorials, and resources. They are trying out Karakeep for its advanced features and want to bring their entire Raindrop.io library with them without a painful manual migration.
   - **Sam, the Productivity Enthusiast**: Sam uses multiple tools and wants to centralize their knowledge base in Karakeep. They need a "fire-and-forget" tool to import their bookmarks so they can focus on their work.
### 3.3 Role-based access
      - **User**: This is the only role. The user must provide their own API tokens for both Raindrop.io (for read access) and Karakeep (for write access) to authorize the tool to perform the import on their behalf.

## 4. Functional requirements
   - **Authentication with APIs** (Priority: High)
     - Read the Raindrop.io API token from an environment variable.
     - Read the Karakeep API token from an environment variable.
   - **Fetch Bookmarks from Raindrop.io** (Priority: High)
     - Fetch all bookmarks from the user's Raindrop.io account.
     - Handle pagination in the Raindrop.io API to retrieve all records.
     - Extract relevant metadata: URL, title, description/excerpt, tags, and creation date.
   - **Create Bookmarks in Karakeep** (Priority: High)
     - Create new bookmarks in the user's Karakeep account.
     - Map Raindrop.io data fields to the corresponding Karakeep bookmark fields.
     - Attach tags to the newly created bookmarks in Karakeep.
   - **Rate Limiting Handling** (Priority: High)
     - Respect API rate limits for both Raindrop.io and Karakeep.
     - Implement an exponential backoff strategy to automatically retry failed requests due to rate limiting.
   - **User Feedback and Reporting** (Priority: Medium)
     - Display a progress bar or status updates during the import process.
     - Provide a summary report after the import is complete, showing the number of successful imports and any failures.
     - Log detailed errors to a file for troubleshooting purposes.
   - **Configuration** (Priority: Medium)
     - Provide a command-line interface (CLI) for running the tool.
     - Support configuration via environment variables or a `.env` file for API keys and other settings.

## 5. User experience
### 5.1. Entry points & first-time user flow
   - The user downloads the tool and runs it from their terminal.
   - The user configures the necessary environment variables (`RAINDROP_API_TOKEN`, `KARAKEEP_API_TOKEN`) in their shell or through a `.env` file.
   - If the environment variables are not set, the tool will exit with a clear error message instructing the user on how to set them.
### 5.2. Core experience
   - **Run the tool**: The user executes the import command from their terminal.
     - The tool provides clear and immediate feedback that the process has started.
   - **Fetch and process**: The tool fetches bookmarks from Raindrop.io, showing a progress indicator.
     - The user should feel confident that the tool is working and not stalled.
   - **Import and report**: The tool creates bookmarks in Karakeep, handling any API limits gracefully, and provides a final summary.
     - The user gets a clear report of what was accomplished and if any actions are needed.
### 5.3. Advanced features & edge cases
   - The tool should detect and handle potential duplicate bookmarks if the import is run multiple times.
   - If an API token is invalid or expired, the tool should provide a clear error message and prompt the user to re-configure.
   - The tool should gracefully handle network errors and resume where possible.
### 5.4. UI/UX highlights
   - A clean, simple, and informative command-line interface.
   - A dynamic progress bar that provides real-time feedback on the import status.
   - Clear, concise, and actionable error messages.

## 6. Narrative
Alex is a dedicated researcher who has curated thousands of bookmarks in Raindrop.io over the years. They've recently discovered Karakeep and are excited by its powerful knowledge management features, but the thought of manually transferring every single bookmark is daunting. Alex finds this tool, a simple command-line utility. After a quick configuration process where they provide their API keys, they run the import. The tool displays a progress bar and informs them as it fetches data from Raindrop and populates Karakeep, even pausing and retrying when it hits an API rate limit. In a few minutes, their entire bookmark library is available in Karakeep, ready to be organized and utilized, saving them hours of tedious work.

## 7. Success metrics
### 7.1. User-centric metrics
   - High successful import completion rate (>99%).
   - Low number of user-reported issues or bugs.
   - Positive feedback and high ratings on platforms like GitHub.
### 7.2. Business metrics
   - Measurable increase in bookmark creation in Karakeep via the API.
   - Mentions of the tool in online communities as a key reason for switching to Karakeep.
### 7.3. Technical metrics
   - Low API error rates, particularly for rate-limiting events.
   - Efficient handling of large bookmark collections with minimal performance degradation.
   - High test coverage for core import and API handling logic.

## 8. Technical considerations
### 8.1. Integration points
   - Raindrop.io REST API for fetching bookmarks.
   - Karakeep REST API for creating bookmarks and tags.
### 8.2. Data storage & privacy
   - User API tokens will be read from environment variables and will not be stored by the tool.
   - The tool will not store or transmit any bookmark data or personal information to any third-party service.
### 8.3. Scalability & performance
   - The application must handle large numbers of bookmarks by using pagination when fetching from the Raindrop.io API.
   - It must implement robust rate-limiting handling (e.g., exponential backoff) to avoid overwhelming the Karakeep API and ensure the import process completes successfully.
### 8.4. Potential challenges
   - Potential differences in data models between Raindrop.io and Karakeep that may require complex data transformation logic.
   - Keeping the tool updated in response to changes in either the Raindrop.io or Karakeep APIs.
   - Ensuring the secure storage of user API tokens across different operating systems.

### 8.5. Data Fetching Strategy
   - The tool will use the Raindrop.io REST API to fetch all user data, including bookmarks, collections, and tags.
   - A full data dump will be achieved by making a series of paginated requests to the relevant API endpoints.
   - The backup/export feature of Raindrop.io will not be used, as it does not provide a seamless, real-time user experience.

## 9. Milestones & sequencing
### 9.1. Project estimate
   - Small: 1-2 weeks for a functional command-line tool.
### 9.2. Team size & composition
   - Small Team: 1 engineer.
### 9.3. Suggested phases
   - **Phase 1**: Core CLI Functionality (1 week)
     - Key deliverables: Basic CLI to accept API keys as arguments, fetch all bookmarks from Raindrop, and create them in Karakeep. Basic error handling and progress display.
   - **Phase 2**: Improved Configuration and Error Handling (1 week)
     - Key deliverables: Support for `.env` files, implement robust rate-limiting with exponential backoff, provide a summary report of the import.

## 10. User stories
### 10.1. Configure API access
   - **ID**: US-001
   - **Description**: As a user, I want to securely provide my Raindrop.io and Karakeep API tokens through environment variables so the tool can access my accounts.
   - **Acceptance criteria**:
     - The tool reads the `RAINDROP_API_TOKEN` and `KARAKEEP_API_TOKEN` from environment variables.
     - If a `.env` file is present in the project directory, the tool loads the variables from it.
     - If the required environment variables are not set, the tool exits with an informative error message.
     - The tool documentation clearly explains how to set the required environment variables.
### 10.2. Perform a full import
   - **ID**: US-002
   - **Description**: As a user, I want to import all my bookmarks from Raindrop.io to Karakeep so I can migrate my data in one go.
   - **Acceptance criteria**:
     - The tool fetches all bookmarks from the user's Raindrop.io account, handling pagination correctly.
     - For each Raindrop.io bookmark, a corresponding bookmark is created in Karakeep.
     - The bookmark's URL, title, description, and tags are correctly transferred.
### 10.3. See import progress
   - **ID**: US-003
   - **Description**: As a user, I want to see the progress of the import process so I know the tool is working and how long it might take.
   - **Acceptance criteria**:
     - A progress bar is displayed in the terminal during the import process.
     - The progress bar updates in real-time to reflect the number of bookmarks imported versus the total.
     - The tool prints status messages indicating the current stage (e.g., "Fetching from Raindrop.io", "Importing to Karakeep").
### 10.4. Get an import summary
   - **ID**: US-004
   - **Description**: As a user, I want to receive a summary after the import is finished so I can verify how many bookmarks were transferred successfully and if there were any errors.
   - **Acceptance criteria**:
     - After the import completes, a summary is printed to the console.
     - The summary includes the total number of bookmarks successfully imported.
     - The summary includes the total number of bookmarks that failed to import.
     - If there were failures, the summary provides information on where to find detailed error logs.
### 10.5. Handle API limits gracefully
   - **ID**: US-005
   - **Description**: As a user with many bookmarks, I want the tool to handle API rate limits automatically so that the entire import process doesn't fail.
   - **Acceptance criteria**:
     - When a rate limit error is received from an API, the tool does not exit.
     - The tool waits for a specified period before retrying the failed request.
     - The retry attempts use an exponential backoff strategy to avoid overwhelming the API.
     - The import process continues from where it left off after a successful retry.
</prd_outline>
