import { expect, test } from "../test-setup";

test("navigate with hash in file name", async({ page, checkForErrors }) => {
  await page.goto("/files/");
  await expect(page).toHaveTitle("Graham's Filebrowser - Files - playwright-files");
  await page.locator('a[aria-label="folder#hash"]').waitFor({ state: 'visible' });
  await page.locator('a[aria-label="folder#hash"]').dblclick();
  await expect(page).toHaveTitle("Graham's Filebrowser - Files - folder#hash");
  await page.locator('a[aria-label="file#.sh"]').waitFor({ state: 'visible' });
  await page.locator('a[aria-label="file#.sh"]').dblclick();
  await expect(page).toHaveTitle("Graham's Filebrowser - Files - file#.sh");
  await expect(page.locator('.topTitle')).toHaveText('file#.sh');
  checkForErrors()
})

test("breadcrumbs display checks", async({ page, checkForErrors }) => {
  await page.goto("/files/playwright%20+%20files/myfolder");
  await page.waitForSelector('#breadcrumbs');
  let spanChildrenCount = await page.locator('#breadcrumbs > ul > li.item').count();
  expect(spanChildrenCount).toBe(1);
  const homeBreadcrumb = page.locator('#breadcrumbs > ul > li:first-child .breadcrumb-link');
  await expect(homeBreadcrumb).toHaveCount(1);
  let breadCrumb = page.locator('span.breadcrumb-link[aria-label="breadcrumb-link-myfolder"]')
  await expect(breadCrumb).toHaveText("myfolder");
  await expect(page.locator('a[aria-label="breadcrumb-link-myfolder"]')).toHaveCount(0);
  const currentUrl = page.url();
  await homeBreadcrumb.click();
  await breadCrumb.click();
  await expect(page).toHaveURL(currentUrl);

  await page.goto("/files/playwright%20+%20files/myfolder/testdata");
  await page.waitForSelector('#breadcrumbs');
  spanChildrenCount = await page.locator('#breadcrumbs > ul > li.item').count();
  expect(spanChildrenCount).toBe(2);
  breadCrumb = page.locator('span.breadcrumb-link[aria-label="breadcrumb-link-testdata"]')
  await expect(breadCrumb).toHaveText("testdata");
  await expect(page.locator('a[aria-label="breadcrumb-link-testdata"]')).toHaveCount(0);

  await page.goto("/files/playwright%20+%20files/files");
  await page.waitForSelector('#breadcrumbs');
  spanChildrenCount = await page.locator('#breadcrumbs > ul > li.item').count();
  expect(spanChildrenCount).toBe(1);
  breadCrumb = page.locator('span.breadcrumb-link[aria-label="breadcrumb-link-files"]')
  await expect(breadCrumb).toHaveText("files");
  await expect(page.locator('a[aria-label="breadcrumb-link-files"]')).toHaveCount(0);
  checkForErrors();
});

test("navigate from search item", async({ page, checkForErrors }) => {
  await page.goto("/files/");
  await expect(page).toHaveTitle("Graham's Filebrowser - Files - playwright-files");
  await page.locator('#search-bar-input').click()
  await page.locator('#search-input').fill('for testing');
  await expect(page.locator('#result-list')).toHaveCount(1);
  await page.locator('li[aria-label="for testing.md"]').click();
  await expect(page).toHaveTitle("Graham's Filebrowser - Files - for testing.md");
  await expect(page.locator('.topTitle')).toHaveText('for testing.md');
  checkForErrors()
});

test("use quick jump", async({ page, checkForErrors }) => {
  await page.goto("/files/playwright%20%2B%20files/myfolder/testdata/gray-sample.jpg");
  await expect(page).toHaveTitle("Graham's Filebrowser - Files - gray-sample.jpg");

  // drag next button to the left to open quick jump list
  const nextButton = page.locator('button[aria-label="Next"]');
  await nextButton.waitFor({ state: "visible" });
  const box = await nextButton.boundingBox();
  if (!box) throw new Error("Could not get bounding box of Next button");
  const startX = box.x + box.width / 2;
  const startY = box.y + box.height / 2;
  await page.mouse.move(startX, startY);
  await page.mouse.down();
  await page.mouse.move(startX - 200, startY);
  await page.mouse.up();

  const quickJumpWindow = page.locator('div.floating-window[aria-label="file-list-prompt"]');
  await expect(quickJumpWindow).toBeVisible();
  await page.locator('div[aria-label="20130612_142406.jpg"]').click();
  await expect(page).toHaveTitle("Graham's Filebrowser - Files - 20130612_142406.jpg");
  await expect(page.locator('.topTitle')).toHaveText('20130612_142406.jpg');
  await expect(quickJumpWindow).toBeHidden();
  checkForErrors();
})
