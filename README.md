# Go DNS Resolver (in a weekend)
This is a failed attempt at following this blog post: https://implement-dns.wizardzines.com/book/intro.html

They have a pretty nice guide on implementing a DNS resolver from scratch. They did it in Python and I tried to do it in Go. I spent several hours with ChatGPT trying to understand and fix the issues and never got it :(

## The Failure
After sending a DNS request to Google's nameservers and getting a response, the task is to parse the response into something meaningful. This program fails to parse Records that should be in the response after the Header and Questions, both things that are correctly parsed.

In the blog post guide I was following, they use a Python BytesIO buffer reader to operate on the response data. This is convenient because the pointer in the buffer is advanced as data is read, so functions can be chained together and benefit from pointer movement the previous function did.

This [confounding snippet](https://implement-dns.wizardzines.com/book/part_2.html#finish-our-dnsrecord-parsing) was shown:

In
```python
reader = BytesIO(response)
parse_header(reader)
parse_question(reader)
parse_record(reader)
```
Out:
```console
DNSRecord(name=b'www.example.com', type_=1, class_=1, ttl=21147, data=b']\xb8\xd8"')
```

The issue is that `parse_question` and `parse_record` both call `decode_name`, which would have advanced the buffer pointer past the name bytes in the response. So by the time `parse_record` is called, the pointer has moved too far and parsing starts to fail and get garbage data into the DNSRecord struct.

I was excited for the blog post but disappointed by the lack of explanation of the code. This disappointment was compounded by the fact that ChatGPT was unable to fix this rather simple program after quite a bit of back and forth. Womp womp.