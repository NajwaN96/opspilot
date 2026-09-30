/** Public AWS build marker. Local development leaves this unset. */
export function isPortfolioRuntime(value: string | undefined = process.env.NEXT_PUBLIC_OPSPILOT_RUNTIME): boolean {
  return value === "aws-portfolio-demo";
}
