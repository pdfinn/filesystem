# Changelog

## [1.0.1] - 2025-05-28

### Fixed
- **Build Issues**: Fixed compilation errors that prevented the server from building
  - Removed redundant `pkg/filesystem/fileinfo_other.go` file that was causing struct redeclaration conflicts on Linux
  - Fixed `buildTree` method signature to include missing `depth` parameter
  - Fixed undefined `rename` function by using `os.Rename`
  - Updated GitHub Actions workflow to use Go 1.24.2

- **Test Fixes**: Resolved multiple test failures and improved test reliability
  - Fixed `getEditOperations` parameter parsing to handle correct slice types (`[]interface{}` vs `[]map[string]interface{}`)
  - Updated handler tests to check for MCP error results instead of Go errors (correct MCP pattern)
  - Fixed path validation tests to handle symlink resolution on macOS (`/var` vs `/private/var`)
  - Updated test expectations to be more flexible with path resolution

- **Path Validation**: Enhanced path security and compatibility on macOS
  - Improved `NewPathValidator` to resolve symlinks for allowed directories 
  - Added both original and resolved paths to allowed directories list for better compatibility
  - Fixed symlink handling in temporary directory tests

- **Security**: Maintained strict security validation while improving usability
  - All path validation tests passing
  - Cross-device move operations properly handled
  - Symlink loop detection working correctly

### Improved
- **Logging**: All operations use structured logging with proper levels
- **Error Handling**: MCP handlers return proper error results instead of Go errors
- **Test Coverage**: All 25+ tests now passing with improved assertions
- **Code Quality**: Follows sound software engineering principles and security best practices

### Verified
- ✅ All 11 filesystem tools working correctly
- ✅ All tests passing (cmd, handlers, config, filesystem, security)
- ✅ Server starts and runs successfully
- ✅ Proper MCP protocol compliance
- ✅ Security validation working
- ✅ Claude Desktop integration ready

## [1.0.0] - 2025-05-27

### Added
- Initial release of secure filesystem MCP server
- 11 filesystem tools with comprehensive security validation
- Support for macOS, Linux, and Windows platforms
- Structured logging with slog
- Comprehensive test suite
- Claude Desktop integration support 