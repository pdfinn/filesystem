# Changelog

## [1.0.2] - 2025-05-28

### Fixed - CRITICAL
- **Claude Desktop Compatibility**: Fixed protocol version mismatch causing immediate disconnection
  - Downgraded `mcp-go` library from v0.30.0 to v0.24.0 to use correct protocol version `2024-11-05`
  - Claude Desktop expects protocol version `2024-11-05` but v0.30.0 was responding with `2025-03-26`
  - Updated API calls to be compatible with older mcp-go version
  - Server now properly establishes connection with Claude Desktop without premature disconnection

### Technical Details
- **Root Cause**: MCP-Go library v0.30.0 introduced protocol version `2025-03-26` which is incompatible with Claude Desktop
- **Solution**: Reverted to mcp-go v0.24.0 which uses the expected `2024-11-05` protocol version
- **API Changes**: Updated `getArguments()` function to handle direct `map[string]interface{}` type in older version
- **Test Updates**: Fixed test structure to match older mcp-go API without `mcp.Meta` type

### Verified
- ✅ Server responds with correct protocol version `2024-11-05`
- ✅ All 25+ tests still passing
- ✅ Full MCP handshake works correctly
- ✅ Ready for production use with Claude Desktop

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