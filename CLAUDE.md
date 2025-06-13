# RainBridge Development Guide

## Build, Test, and Lint Commands
- Build: `npm run build` 
- Lint: `npm run lint`
- Test: `npm run test`
- Test single file: `npm run test -- path/to/test-file.ts`
- Dev mode: `npm run dev`

## Code Style Guidelines
- **Formatting**: Use Prettier with default settings
- **Linting**: ESLint with TypeScript rules
- **Naming**:
  - camelCase for variables and functions
  - PascalCase for classes and types
  - UPPER_SNAKE_CASE for constants
- **Imports**: Group imports (React/external/internal) with a blank line between groups
- **Types**: Prefer explicit typing over `any`; use TypeScript interfaces for API responses
- **Error Handling**: Use try/catch blocks with appropriate logging; implement exponential backoff for API calls
- **APIs**: Implement rate limiting and throttling for Raindrop and Karakeep API calls

## Project Structure
This utility imports bookmarks from Raindrop.io into Karakeep using their respective APIs.