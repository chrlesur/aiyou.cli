
# Requirements Specification: AI.YOU CLI Client

## 1. Context

The AI.YOU CLI is a command-line client for the AI.YOU API, leveraging the official aiyou.golib library. It aims to provide a robust, efficient, and user-friendly interface for interacting with AI assistants, managing conversations, and performing various AI-related tasks.

## 2. Existing Functionalities to Maintain

### 2.1 Authentication
- Login to the AI.YOU API with email and password
- Management of authentication tokens

### 2.2 Interaction with Assistants
- Sending text messages to a specified assistant
- Support for interactive mode for continuous conversations
- Support for non-interactive mode for one-time queries

### 2.3 Input/Output Management
- Ability to use the application with command-line pipes
- Reading inputs from stdin if no arguments are provided

### 2.4 Configuration
- Loading configuration from a .env file
- Configuration options via command-line flags

### 2.5 Logging and Debugging
- Debug mode to display detailed information
- Quiet mode to limit outputs

## 3. New Functionalities to Add

### 3.1 Audio Support
- Transcription of audio files to text
- Supported formats: MP3, WAV, M4A
- File size limit: 25 MB
- Usage example: `aiyou transcribe audio.mp3 --language en`

### 3.2 Advanced Thread Management
- Creation, retrieval, and deletion of conversation threads
- Listing of existing threads with pagination
- Usage example: `aiyou list-threads --page 1 --items-per-page 10`

### 3.3 Response Streaming
- Option to receive assistant responses in streaming mode
- Real-time display of partial responses
- Usage example: `aiyou chat --stream "What is the capital of France?"`

### 3.4 Advanced Model Parameters
- Options to adjust model parameters:
  - Temperature (range: 0.0 to 1.0)
  - Top_p (range: 0.0 to 1.0)
  - Maximum number of tokens (according to model limits)
- Usage example: `aiyou chat --temperature 0.7 --top-p 0.9 --max-tokens 100 "Generate a short story"`

### 3.5 Assistant Management
- Listing of available assistants
- Selection of an assistant for conversation
- Usage example: `aiyou list-assistants` and `aiyou chat --assistant-id "asst_123" "Hello"`

### 3.6 Caching System
- Implementation of an in-memory cache for frequently accessed data
- Cache types: AssistantCache, QueryCache, ModelCache, UserCache, ConfigCache
- Configurable Time-To-Live (TTL) for each cache type
- Thread-safe operations with concurrent access support
- Automatic cleanup of expired entries
- Cache statistics and monitoring

## 4. Technical Requirements

### 4.1 Use of aiyou.golib Library
- Exclusive use of the aiyou.golib library for API interactions
- Adherence to interfaces and structures defined in aiyou.golib

### 4.2 Project Structure
- Clear code organization (cmd, internal, pkg)
- Limitation of file sizes to a maximum of 500 lines
- Limitation of functions to a maximum of 50 lines

### 4.3 Error Handling
- Appropriate handling of API, network, and authentication errors
- Creation of custom error types if necessary
- Consistent use of Go 1.13+ error wrapping

### 4.4 Testing
- Unit tests for each package
- Integration tests with mocks to simulate the API
- Code coverage of at least 80%
- Use of `go test` and `go cover` for tests and coverage

### 4.5 Documentation
- Complete GoDoc documentation for all exported functions
- Detailed README with quick start guide and usage examples
- Documentation of configuration options and commands
- Use of `godoc` to generate documentation

### 4.6 Performance
- Implementation of a rate limiter to respect API limits
- Performance optimization for frequent operations
- Benchmarking of critical operations with `go test -bench`

### 4.7 Security
- Secure storage of authentication tokens
- Encryption of sensitive data in transit and at rest
- Validation and escaping of user inputs to prevent injections

### 4.8 Caching
- Implementation of a thread-safe in-memory cache
- Support for different cache types with configurable TTLs
- Automatic cleanup of expired cache entries
- Cache size limitations and eviction policies
- Cache statistics for monitoring and optimization

## 5. Legal and Publication Constraints

### 5.1 License
- Use of the GNU General Public License v3.0 (GPL-3.0)
- Inclusion of the full license text in the project
- Addition of a license header in each source file

### 5.2 Contact Information
- Main maintainer: GitHub login "chrlesur"
- Contact email: christophe.lesur@cloud-temple.com

### 5.3 Publication
- Publication of the code on GitHub
- Creation of versioned releases following the SemVer model

## 6. Integration with Existing Code

### 6.1 User Interface Modifications
- Adaptation of existing commands to integrate new functionalities
- Maintaining backward compatibility with scripts using the old version

### 6.2 Functionality Migration
- Detailed migration plan for each existing functionality to use aiyou.golib

## 7. Scalability

### 7.1 Extensible Architecture
- Modular design allowing easy addition of new commands
- Use of interfaces for key components to facilitate future extensions

### 7.2 Extension Points
- Plugin system to allow addition of custom functionalities
- Documented internal API for third-party developers

## 8. Alignment with Go Standards

### 8.1 Code Conventions
- Strict adherence to Go naming conventions (https://golang.org/doc/effective_go.html#names)
- Use of `gofmt` for automatic code formatting

### 8.2 Code Quality Tools
- Integration of `golint` and `go vet` in the development process
- Configuration of GitHub Actions to run these tools on each push

## 9. Glossary

- **Thread**: A sequence of messages between the user and the assistant
- **Assistant**: A specific AI model with defined capabilities and knowledge
- **Streaming**: Method of receiving responses in real-time chunks
- **Token**: Text unit used by AI models to process language
- **Cache**: Temporary storage of frequently accessed data to improve performance
- **TTL (Time-To-Live)**: The duration for which a cache entry is considered valid
