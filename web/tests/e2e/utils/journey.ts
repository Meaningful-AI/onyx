import { mkdir, writeFile } from "fs/promises";
import path from "path";

import type { Page } from "@playwright/test";

function captureDir(): string | null {
  const value = process.env.PLAYWRIGHT_JOURNEY_CAPTURE_DIR;
  if (!value) {
    return null;
  }
  return value;
}

function slug(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export async function captureJourneyCheckpoint(
  page: Page,
  name: string
): Promise<void> {
  const dir = captureDir();
  if (!dir) {
    return;
  }

  const checkpoint = slug(name) || "checkpoint";
  await mkdir(dir, { recursive: true });

  const screenshotPath = path.join(dir, `${checkpoint}.png`);
  const metadataPath = path.join(dir, `${checkpoint}.json`);

  await page.screenshot({ path: screenshotPath, fullPage: true });
  await writeFile(
    metadataPath,
    JSON.stringify(
      {
        checkpoint,
        url: page.url(),
        title: await page.title(),
        captured_at: new Date().toISOString(),
      },
      null,
      2
    )
  );
}
