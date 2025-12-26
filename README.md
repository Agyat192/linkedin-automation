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

## Prerequisites

- Go 1.21 or higher
- Google Chrome or Microsoft Edge browser
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

## Configuration Options

### LinkedIn Credentials
```yaml
linkedin:
  email: "your-email@example.com"    # Your LinkedIn email
  password: "your-password"         # Your LinkedIn password
```

### Browser Settings
```yaml
browser:
  headless: true                    # Run browser in background
  user_agent: "custom-string"       # Custom user agent
  disable_images: false             # Disable images for faster loading
```

### Rate Limits
```yaml
limits:
  daily_connections: 50             # Max connections per day
  hourly_connections: 10             # Max connections per hour
  cooldown_period: "30m"            # Wait time between batches
```

## Troubleshooting

### Common Issues

#### "LinkedIn checkpoint verification required"
- Solution: Complete verification in the browser window that opens
- Alternative: Log into LinkedIn manually first, then run automation

#### "CAPTCHA challenge detected"
- Solution: Solve CAPTCHA in the browser window
- Tip: Use `--headless=false` from the start

#### "Failed to connect to existing browser"
- Solution: Close other Chrome instances
- Alternative: Let the application launch a new browser

#### "Context canceled" errors
- Solution: Usually normal when operations complete quickly
- Action: Check if the operation actually succeeded

### Debug Mode

Enable verbose logging for troubleshooting:
```bash
./linkedin-automation search users --keywords "Developer" --verbose
```

## Security Considerations

- **Store credentials securely**: Use environment variables in production
- **Use dedicated account**: Consider using a separate LinkedIn account
- **Monitor usage**: Track your automation activity
- **Follow LinkedIn policies**: Respect LinkedIn's terms of service

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is for educational purposes. Use responsibly and in accordance with LinkedIn's terms of service.

## Disclaimer

This tool is for educational purposes only. Users are responsible for:
- Complying with LinkedIn's terms of service
- Ensuring lawful use of the automation
- Any consequences of misuse

Use at your own risk and respect LinkedIn's automation policies.

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review the logs with `--verbose` flag
3. Open an issue on GitHub

---

**Remember**: Automation tools should be used responsibly and ethically. Always respect platform policies and user privacy.
- LinkedIn account with valid credentials

## Installation

1. Clone the repository:
```bash
git clone <your-repo-url>
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

## Configuration Options

### LinkedIn Credentials
```yaml
linkedin:
  email: "your-email@example.com"    # Your LinkedIn email
  password: "your-password"         # Your LinkedIn password
```

### Browser Settings
```yaml
browser:
  headless: true                    # Run browser in background
  user_agent: "custom-string"       # Custom user agent
  disable_images: false             # Disable images for faster loading
```

### Rate Limits
```yaml
limits:
  daily_connections: 50             # Max connections per day
  hourly_connections: 10             # Max connections per hour
  cooldown_period: "30m"            # Wait time between batches
```

## Troubleshooting

### Common Issues

#### "LinkedIn checkpoint verification required"
- Solution: Complete verification in the browser window that opens
- Alternative: Log into LinkedIn manually first, then run automation

#### "CAPTCHA challenge detected"
- Solution: Solve CAPTCHA in the browser window
- Tip: Use `--headless=false` from the start

#### "Failed to connect to existing browser"
- Solution: Close other Chrome instances
- Alternative: Let the application launch a new browser

#### "Context canceled" errors
- Solution: Usually normal when operations complete quickly
- Action: Check if the operation actually succeeded

### Debug Mode

Enable verbose logging for troubleshooting:
```bash
./linkedin-automation search users --keywords "Developer" --verbose
```

## Security Considerations

- **Store credentials securely**: Use environment variables in production
- **Use dedicated account**: Consider using a separate LinkedIn account
- **Monitor usage**: Track your automation activity
- **Follow LinkedIn policies**: Respect LinkedIn's terms of service

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is for educational purposes. Use responsibly and in accordance with LinkedIn's terms of service.

## Disclaimer

This tool is for educational purposes only. Users are responsible for:
- Complying with LinkedIn's terms of service
- Ensuring lawful use of the automation
- Any consequences of misuse

Use at your own risk and respect LinkedIn's automation policies.

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review the logs with `--verbose` flag
3. Open an issue on GitHub

---

**Remember**: Automation tools should be used responsibly and ethically. Always respect platform policies and user privacy.
=======
# linkedin-automation
>>>>>>> 5770d3092efebeae4906e750f1a5a07b72433d6a
