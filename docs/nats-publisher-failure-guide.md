# NATS Publisher Failure & Retry Handling

## Purpose

This guide defines failure and retry handling for NATS message publishing.

It covers Core NATS and JetStream publishing, including:

* Publish failure classification
* Retryable and non-retryable publish failures
* Retry limits and backoff
* Publish operation deadlines
* Unknown publish outcomes
* JetStream duplicate detection
* JetStream publish expectations

---

# 1. Publish Failure Classification

A publish failure should be classified before deciding whether to retry or fail the publish operation.

```text
Publish
   |
   +-- Success
   |
   +-- Failure
        |
        +-- Permanent Failure ------> Fail
        |
        +-- Transient Failure ------> Retry
        |
        +-- State Conflict ----------> Re-evaluate State
        |
        +-- Unknown Outcome ---------> Apply Duplicate / Idempotency Strategy
```

### 1.1 Permanent Failure

A permanent failure indicates that repeating the same publish without correcting the underlying condition is unlikely to succeed.

Examples:

* Invalid subject or publish request
* Authentication failure
* Authorization / publish permission failure
* Invalid message configuration
* Payload exceeding configured limits

**Action:** Fail the publish operation and correct the underlying issue.

---

### 1.2 Transient Failure

A transient failure represents a temporary condition that may succeed when the publish is attempted again.

Examples:

* Temporary connection failure
* Reconnection-related failure
* Temporary server-side failure
* Temporary JetStream server or storage failure

**Action:** Retry according to the configured retry policy.

---

### 1.3 State Conflict

A state conflict occurs when a JetStream publish depends on a specific stream state that is no longer true.

For example:

```text
Expected Last Sequence = 100
             |
             v
Current Last Sequence = 101
             |
             v
       Publish Rejected
```

This can occur with JetStream publish expectations such as:

* Expected last sequence
* Expected last message ID

**Action:** Do not blindly retry the same publish. Re-evaluate the current stream/application state before issuing another publish.

---

### 1.4 Unknown Publish Outcome

An unknown outcome occurs when the client cannot determine whether JetStream accepted the publish.

For example:

```text
Publisher                              JetStream

    |                                      |
    |------ Publish Message -------------->|
    |                                      |
    |                              Process / Store
    |                                      |
    |<------------- PubAck ----------------X
    |                           Connection Failure
    |
    v
Unknown Outcome
```

The server may have accepted the message even though the client did not receive the acknowledgement.

**Action:** If the same logical message is retried, use an appropriate duplicate-detection or idempotency strategy.

---

# 2. Publish Failure Handling

The failure classification determines the action to take.

| Publish Failure                                          | Classification  | Recommended Action                                 |
| :------------------------------------------------------- | :-------------- | :------------------------------------------------- |
| Invalid subject / request                                | Permanent       | Fail and correct the request                       |
| Authentication failure                                   | Permanent       | Correct client credentials                         |
| Authorization / permission failure                       | Permanent       | Correct publish permissions                        |
| Temporary connection failure                             | Transient       | Retry according to policy                          |
| Temporary JetStream server failure                       | Transient       | Retry according to policy                          |
| Publish acknowledgement timeout                          | Unknown Outcome | Apply duplicate / idempotency strategy             |
| Publish expectation mismatch                             | State Conflict  | Re-evaluate stream state                           |
| Connection failure after publish may have reached server | Unknown Outcome | Retry using stable message identity where required |

A publish should not be retried simply because an error was returned. The application should first determine whether repeating the operation is appropriate.

---

# 3. Retryable Publish Failures

Transient publish failures can be retried when the operation is expected to succeed after the temporary condition clears.

Typical examples include:

* Temporary connection unavailability
* Reconnection-related failures
* Temporary server-side failures
* Temporary JetStream storage/API failures

The retry policy should define:

* Maximum attempts
* Backoff between attempts
* Optional jitter
* Overall operation deadline

A retry implementation should also stop when the operation deadline is reached.

```go
func publishWithRetry(
    ctx context.Context,
    js jetstream.JetStream,
    subject string,
    payload []byte,
    maxAttempts int,
) (*jetstream.PubAck, error) {

    var ack *jetstream.PubAck
    var err error

    backoff := 100 * time.Millisecond

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        ack, err = js.Publish(ctx, subject, payload)

        if err == nil {
            return ack, nil
        }

        if !isRetryable(err) {
            return nil, err
        }

        if attempt == maxAttempts {
            break
        }

        select {
        case <-ctx.Done():
            return nil, ctx.Err()

        case <-time.After(backoff):
            backoff *= 2
        }
    }

    return nil, fmt.Errorf(
        "publish failed after %d attempts: %w",
        maxAttempts,
        err,
    )
}
```

The example demonstrates exponential backoff. The platform retry policy should define the actual backoff strategy and limits.

---

# 4. Non-Retryable Publish Failures

Non-retryable failures represent conditions where repeating the same publish without correcting the underlying issue is unlikely to succeed.

| Failure                               | Reason                                             | Corrective Action                                               |
| :------------------------------------ | :------------------------------------------------- | :-------------------------------------------------------------- |
| **Invalid Subject / Request**         | Target subject or request is invalid               | Correct the subject or request                                  |
| **Authentication Failure**            | Client credentials are invalid                     | Correct client credentials                                      |
| **Authorization / Permission Denied** | Account does not have publish permission           | Correct authorization / ACL configuration                       |
| **Payload Size Limit Exceeded**       | Payload exceeds configured server limit            | Reduce message size or use an appropriate large-payload pattern |
| **Publish Expectation Failure**       | Expected stream state does not match current state | Re-evaluate application and stream state                        |

These failures should not be retried blindly.

The application should fail the publish operation, record sufficient diagnostic context, and correct the underlying condition.

---

# 5. Unknown Publish Outcome

A JetStream publish can have an unknown outcome when the server may have accepted the message but the client does not receive the corresponding `PubAck`.

```text
Publisher                              JetStream

    |                                      |
    |------ Publish Message -------------->|
    |                                      |
    |                         Store Message
    |                                      |
    |<----------- PubAck ------------------X
    |                          Network Failure
    |
    v
Client cannot determine whether
the publish was accepted
```

If the application simply publishes the message again, two copies may be stored if the original publish succeeded.

Therefore:

> **A retry after an unknown publish outcome must consider duplicate detection or application-level idempotency.**

---

# 6. JetStream Duplicate Detection

## 6.1 Stable Message ID

JetStream supports duplicate publish detection using the `Nats-Msg-Id` message header.

```go
msg := nats.NewMsg("orders.created")
msg.Data = payload

// Stable ID for the logical message.
msg.Header.Set("Nats-Msg-Id", "order-12345")

ack, err := js.PublishMsg(ctx, msg)
if err != nil {
    return err
}
```

The message ID should identify the **logical message**, not an individual publish attempt.

---

## 6.2 Reusing the Message ID During Retry

When retrying the same logical JetStream message, retain the same `Nats-Msg-Id`.

```text
Initial Publish
Nats-Msg-Id = order-12345
       |
       X
  Unknown Outcome
       |
       v
Retry Publish
Nats-Msg-Id = order-12345
```

Using a new message ID for the retry represents the retry as a different logical message and does not provide the same duplicate-detection behavior.

---

## 6.3 Duplicate Publish Acknowledgement

If JetStream detects that the message ID has already been processed, the publish acknowledgement can indicate that the publish was a duplicate.

```go
ack, err := js.PublishMsg(ctx, msg)
if err != nil {
    return err
}

if ack.Duplicate {
    // Previous publish with this message ID was already processed.
}
```

Duplicate detection prevents the same logical publish from being stored again within the applicable JetStream duplicate-detection window.

This should not be confused with general application idempotency. Side effects outside JetStream may require their own idempotency mechanism.

---

# 7. JetStream Publish Expectations

JetStream supports publish expectations that allow a publisher to require a specific stream state before accepting a publish.

Examples include:

* Expected last sequence
* Expected last message ID

Conceptually:

```text
Publisher
    |
    | Expected Last Msg ID = order-100
    v
JetStream
    |
    +-- Current Last Msg ID = order-100
    |          |
    |          +--> Accept
    |
    +-- Current Last Msg ID != order-100
               |
               +--> Reject
```

These expectations provide an optimistic concurrency mechanism.

An expectation failure should not normally be treated as a transient failure.

The application should:

```text
Expectation Failure
       |
       v
Re-evaluate Current State
       |
       v
Determine Intended Operation
       |
       v
Publish Again if Appropriate
```

---

# 8. Retry Strategy

A retry policy should prevent unbounded retries and retry storms.

### 8.1 Maximum Attempts

Define an explicit maximum number of publish attempts.

```text
Initial Attempt
      |
      v
   Retry 1
      |
      v
   Retry 2
      |
      v
   Retry N
      |
      v
    Fail
```

---

### 8.2 Backoff

Apply a delay between retry attempts.

Exponential backoff can progressively increase the delay:

```text
Attempt 1 → 100 ms
Attempt 2 → 200 ms
Attempt 3 → 400 ms
Attempt 4 → 800 ms
```

The exact values should be defined by the platform/application retry policy rather than assumed to be NATS defaults.

---

### 8.3 Jitter

When multiple application instances retry after a shared failure, identical retry intervals can cause a thundering-herd effect.

Jitter introduces controlled randomness:

```text
Base Backoff
     +
Random Jitter
     |
     v
Actual Retry Delay
```

Use jitter where multiple publishers may retry the same operation concurrently.

---

### 8.4 Overall Publish Deadline

Retries should be bounded by an overall operation deadline.

For Go:

```go
ctx, cancel := context.WithTimeout(
    context.Background(),
    5*time.Second,
)
defer cancel()
```

The deadline applies to the complete publish operation and its retries:

```text
Publish Attempt
      |
      v
   Backoff
      |
      v
Retry Attempt
      |
      v
   Backoff
      |
      v
Retry Attempt
      |
      v
Deadline Reached
      |
      v
    Stop
```

Other client SDKs provide language-specific mechanisms for implementing publish timeouts and deadlines.

---

# 9. End-to-End Publish Failure Flow

```text
                       Publish
                          |
                          v
                       Success?
                      /       \
                    Yes        No
                    |           |
                   Done         v
                         Classify Failure
                               |
             +-----------------+------------------+
             |                 |                  |
         Permanent        Transient         State Conflict
             |                 |                  |
             v                 v                  v
           Fail          Retry Policy        Re-evaluate
                               |
                               |
                         Unknown Outcome?
                          /           \
                        No             Yes
                        |               |
                        v               v
                      Retry       Dedup / Idempotency
                                        |
                                        v
                                      Retry
```

---

# 10. Publisher Guidelines

1. **Classify the publish failure before retrying.**
2. **Do not retry permanent failures blindly.**
3. **Do not blindly retry JetStream expectation failures.**
4. **Use bounded retry attempts.**
5. **Apply backoff between retry attempts.**
6. **Use jitter where concurrent publishers may retry together.**
7. **Bound the complete retry operation with an appropriate deadline.**
8. **Treat publish acknowledgement timeouts as potentially unknown outcomes.**
9. **Retain the same `Nats-Msg-Id` when retrying the same logical JetStream message and duplicate detection is required.**
10. **Do not treat JetStream duplicate detection as a replacement for application-level idempotency.**
11. **Use the timeout/deadline mechanism provided by the application’s NATS client SDK.**
