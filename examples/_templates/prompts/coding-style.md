# Coding Style Guidelines

## General Principles

- **Prefer readability over cleverness** - Code is read more often than written
- **Write tests for new features** - Test coverage ensures maintainability
- **Keep functions small** - Functions should do one thing well (under 50 lines)
- **Use meaningful names** - Variable and function names should be self-documenting

## Code Organization

- **Single Responsibility** - Each module/class/function has one reason to change
- **DRY (Don't Repeat Yourself)** - Extract common logic into reusable components
- **Separation of Concerns** - Business logic, data access, and presentation are separate

## Documentation

- **Comment the "why", not the "what"** - Code should be self-explanatory
- **Document public APIs** - All exported functions need documentation
- **Keep comments up-to-date** - Outdated comments are worse than no comments

## Error Handling

- **Fail fast** - Validate inputs early and return errors immediately
- **Provide context** - Error messages should explain what went wrong and why
- **Don't ignore errors** - Always handle or explicitly propagate errors

## Testing

- **Test behavior, not implementation** - Tests should survive refactoring
- **One assertion per test** - Each test validates a single behavior
- **Readable test names** - Test names describe the scenario and expected outcome

## Code Review

- **Small, focused changes** - Easier to review and less risky to merge
- **Self-review first** - Review your own code before requesting review
- **Respond to feedback** - Address all review comments or explain why not
