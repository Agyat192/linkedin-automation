# LinkedIn Automation Tool

A powerful Go-based automation tool for LinkedIn user search, connection requests, and messaging. This tool uses the Rod library for browser automation and includes advanced features like stealth mode, CAPTCHA handling, and session management.

## Features

- **User Search**: Search for LinkedIn users by keywords, location, company, and job title
- **Connection Requests**: Send personalized connection requests to found profiles
- **Messaging**: Send messages to your connections
- **Stealth Mode**: Human-like browser behavior to avoid detection
- **CAPTCHA Handling**: Automatic detection and manual CAPTCHA solving
- **Session Management**: Persistent login sessions
- **Rate Limiting**: Built-in protection against LinkedIn rate limits
- **Chrome Profile Integration**: Uses your existing Chrome browser profile


### Authentication System
- Environment-based credentials ✔
- Session cookie persistence ✔
- Login failure detection ✔
- Captcha / 2FA detection ✔

### Search & Targeting
- Search by job, company, location, keywords ✔
- Pagination handling ✔
- Duplicate profile detection ✔

### Connection Requests
- Human-like navigation and clicking ✔
- Personalized notes (≤300 chars) ✔
- Daily request limits ✔

### Messaging System
- Accepted connection detection ✔
- Template-based follow-up messages ✔
- Message tracking & persistence ✔

### Anti-Bot & Stealth Techniques
Mandatory:
- Human-like mouse movement (Bezier curves) ✔
- Randomized timing patterns ✔
- Browser fingerprint masking ✔

Additional:
- Random scrolling behavior ✔
- Human typing simulation ✔
- Mouse hovering & wandering ✔
- Activity scheduling ✔
- Rate limiting & throttling ✔

### Code Quality
- Modular architecture ✔
- Structured logging ✔
- Config management ✔
- Persistent state storage ✔

## Prerequisites

- Go 1.21 or higher
- Google Chrome 
- LinkedIn account with valid credentials

## Installation

1. Clone the repository:
```bash
git clone https://github.com/Agyat192/linkedin-automation.git
cd linkedin-automation
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o linkedin-automation
```

## Configuration

The application uses a YAML configuration file. Create `config/config.yaml`:

```yaml
# LinkedIn Credentials
linkedin:
  email: "your-email@example.com"
  password: "your-password"

# Browser Settings
browser:
  headless: true
  user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

# Rate Limiting
limits:
  daily_connections: 50
  hourly_connections: 10
  daily_messages: 100
  hourly_messages: 20
  search_results: 100
  cooldown_period: "30m"

# Storage
storage:
  session_path: "./sessions"
  output_path: "./data"
```

## Usage

### Basic Commands

#### Search Users
```bash
# Search for software engineers in India
./linkedin-automation search users --keywords "Software Engineer" --location "India" --max-results 10

# Search with company filter
./linkedin-automation search users --keywords "Developer" --company "Microsoft" --location "USA"

# Search with job title filter
./linkedin-automation search users --title "Senior Developer" --location "New York" --max-results 20
```

#### Send Connection Requests
```bash
# Send requests to found profiles
./linkedin-automation connect --input "search_results.json" --message "Hi, I'd like to connect with you!"

# Send requests with custom message template
./linkedin-automation connect --input "profiles.json" --message "Hello {{name}}, I found your profile interesting!"
```

#### Send Messages
```bash
# Send messages to existing connections
./linkedin-automation message --input "connections.json" --message "Hi! How are you doing?"

# Send personalized messages
./linkedin-automation message --input "connections.json" --message "Hi {{name}}, saw your post about {{topic}}"
```

### Advanced Options

#### Browser Mode
```bash
# Run with visible browser (useful for CAPTCHA)
./linkedin-automation search users --keywords "Developer" --headless=false

# Verbose logging
./linkedin-automation search users --keywords "Developer" --verbose
```

#### Output Options
```bash
# Save results to file
./linkedin-automation search users --keywords "Developer" --output "results.json"

# Use custom config
./linkedin-automation search users --keywords "Developer" --config "./custom-config.yaml"
```

## Authentication and Security

### LinkedIn Checkpoint Verification

LinkedIn may require additional verification when using automation tools. When this happens:

1. The application will open a browser window
2. Complete LinkedIn's security verification (email/phone)
3. The automation will continue automatically

### CAPTCHA Handling

The application automatically detects CAPTCHA challenges:
- Switches to non-headless mode
- Opens browser window for manual solving
- Continues automation after CAPTCHA is solved

### Chrome Profile Integration

The application uses your existing Chrome profile to:
- Maintain existing LinkedIn sessions
- Reduce detection risk
- Provide seamless authentication

## Rate Limiting and Best Practices

To avoid account suspension:

1. **Start Slow**: Begin with low connection/message limits
2. **Use Realistic Intervals**: Don't send too many requests quickly
3. **Personalize Messages**: Avoid generic, spam-like content
4. **Monitor Your Account**: Watch for LinkedIn warnings
5. **Respect LinkedIn Terms**: Follow LinkedIn's automation policies


**Remember**: Automation tools should be used responsibly and ethically. Always respect platform policies and user privacy.
=======

