# Supabase Realtime Go Client Examples

This directory contains examples demonstrating how to use the Supabase Realtime Go client.

## Examples

1. **basic/main.go**: Basic connection and channel creation example
2. **broadcast/main.go**: Example of using the Broadcast feature (instant messaging, cursor position sharing, etc.)
3. **presence/main.go**: Example of using the Presence feature (user status tracking)
4. **postgres_cdc/main.go**: Example of using the PostgreSQL Change Data Capture (CDC) feature

## Running

To run each example, you first need to update your Supabase project reference and API key in the respective file:

```go
// Replace with your Supabase project reference and API key
projectRef := "your-project-ref"
apiKey := "your-api-key"
```

Then you can run the example as follows:

```bash
go run examples/basic/main.go
```

## Notes

- You need a valid Supabase project to run the examples.
- To test Broadcast and Presence features, you may need to run multiple clients.
- For the PostgreSQL CDC example, you need to create the relevant table in your Supabase project and enable the Realtime feature.
