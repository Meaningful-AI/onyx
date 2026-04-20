import { INTERNAL_URL } from "@/lib/constants";
import { NextRequest, NextResponse } from "next/server";

export async function POST(request: NextRequest) {
  const backendUrl = `${INTERNAL_URL}/agents/wizard-chat`;
  const headers = new Headers();
  let cookieHeader = request.headers.get("cookie") || "";
  if (
    process.env.DEBUG_AUTH_COOKIE &&
    process.env.NODE_ENV === "development"
  ) {
    const debugCookie = `fastapiusersauth=${process.env.DEBUG_AUTH_COOKIE}`;
    cookieHeader = cookieHeader
      ? `${cookieHeader}; ${debugCookie}`
      : debugCookie;
  }
  if (cookieHeader) {
    headers.set("cookie", cookieHeader);
  }
  const contentType = request.headers.get("content-type");
  if (contentType) {
    headers.set("content-type", contentType);
  }

  const backendResponse = await fetch(backendUrl, {
    method: "POST",
    headers,
    body: await request.text(),
  });

  const responseHeaders = new Headers(backendResponse.headers);
  responseHeaders.delete("set-cookie");

  if (
    backendResponse.headers.get("Transfer-Encoding") === "chunked" ||
    backendResponse.headers.get("Content-Type")?.includes("stream") ||
    backendResponse.headers.get("Content-Type")?.includes("text/event-stream")
  ) {
    const { readable, writable } = new TransformStream();
    backendResponse.body?.pipeTo(writable);
    return new NextResponse(readable, {
      status: backendResponse.status,
      headers: responseHeaders,
    });
  }

  return new NextResponse(backendResponse.body, {
    status: backendResponse.status,
    headers: responseHeaders,
  });
}
