import type { Metadata } from "next";
import Link from "next/link";
import { LegalPage } from "@/components/legal-page";
import { SUPPORT_EMAIL, SUPPORT_EMAIL_URL } from "@/lib/site-links";

export const metadata: Metadata = {
  title: "Privacy Policy — gx",
  description: "Privacy Policy for gx by Satori Engineering Co.",
};

export default function PrivacyPage() {
  return (
    <LegalPage title="Privacy Policy" updated="July 14, 2026">
      <section>
        <h2>1. Overview</h2>
        <p>
          This Privacy Policy explains how Satori Engineering Co. (“Satori,”
          “we,” “us,” or “our”) collects, uses, and shares information when you
          use gx and related websites, apps, and services (the “Service”). By
          using the Service, you agree to this policy.
        </p>
      </section>

      <section>
        <h2>2. Information we collect</h2>
        <p>We may collect:</p>
        <ul>
          <li>
            <strong>Account information</strong> — such as name, email address,
            and authentication details when you sign in (for example via GitHub)
          </li>
          <li>
            <strong>Usage and device data</strong> — such as pages viewed,
            product interactions, approximate location derived from IP address,
            browser or app type, and diagnostic logs
          </li>
          <li>
            <strong>Content you provide</strong> — such as repositories you
            connect, code diffs, review comments, and support messages
          </li>
          <li>
            <strong>Payment information</strong> — billing details processed by
            our payment provider (we do not store full card numbers on our
            servers)
          </li>
        </ul>
      </section>

      <section>
        <h2>3. How we use information</h2>
        <p>We use information to:</p>
        <ul>
          <li>Provide, operate, and improve the Service</li>
          <li>Authenticate users and secure accounts</li>
          <li>Process subscriptions and payments</li>
          <li>Communicate about the Service, including support and updates</li>
          <li>Monitor performance, prevent abuse, and debug issues</li>
          <li>Comply with legal obligations</li>
        </ul>
      </section>

      <section>
        <h2>4. Analytics and cookies</h2>
        <p>
          We use analytics tools (such as Google Analytics and PostHog) and
          similar technologies to understand how the Service is used. These tools
          may use cookies or similar identifiers. You can control cookies through
          your browser settings; some features may not work if you disable them.
        </p>
      </section>

      <section>
        <h2>5. How we share information</h2>
        <p>We may share information with:</p>
        <ul>
          <li>
            <strong>Service providers</strong> who help us operate the Service
            (hosting, authentication, payments, analytics, email) under
            obligations to protect your data
          </li>
          <li>
            <strong>Integrations you enable</strong>, such as GitHub, to the
            extent needed for those features
          </li>
          <li>
            <strong>Legal and safety</strong> recipients when required by law or
            to protect rights, safety, and the Service
          </li>
          <li>
            <strong>Business transfers</strong> in connection with a merger,
            acquisition, or sale of assets
          </li>
        </ul>
        <p>We do not sell your personal information.</p>
      </section>

      <section>
        <h2>6. Data retention</h2>
        <p>
          We retain information for as long as needed to provide the Service,
          meet legal obligations, resolve disputes, and enforce our agreements.
          You may request deletion of your account by contacting us; some data
          may remain in backups or logs for a limited period.
        </p>
      </section>

      <section>
        <h2>7. Security</h2>
        <p>
          We use reasonable technical and organizational measures to protect
          information. No method of transmission or storage is completely secure,
          and we cannot guarantee absolute security.
        </p>
      </section>

      <section>
        <h2>8. International transfers</h2>
        <p>
          We may process information in the United States and other countries
          where we or our providers operate. Those locations may have different
          data protection laws than your home country.
        </p>
      </section>

      <section>
        <h2>9. Your choices</h2>
        <p>Depending on where you live, you may have rights to:</p>
        <ul>
          <li>Access, correct, or delete personal information</li>
          <li>Object to or restrict certain processing</li>
          <li>Request a copy of your data</li>
        </ul>
        <p>
          To exercise these rights, email{" "}
          <a href={SUPPORT_EMAIL_URL}>{SUPPORT_EMAIL}</a>. We may need to verify
          your request.
        </p>
      </section>

      <section>
        <h2>10. Children</h2>
        <p>
          The Service is not directed to children under 16, and we do not
          knowingly collect personal information from them. If you believe a
          child has provided us information, contact us and we will take
          appropriate steps.
        </p>
      </section>

      <section>
        <h2>11. Changes</h2>
        <p>
          We may update this Privacy Policy from time to time. We will post the
          updated policy on this page and revise the “Last updated” date.
          Continued use after changes become effective constitutes acceptance of
          the updated policy.
        </p>
      </section>

      <section>
        <h2>12. Contact</h2>
        <p>
          Privacy questions:{" "}
          <a href={SUPPORT_EMAIL_URL}>{SUPPORT_EMAIL}</a>
        </p>
        <p>
          See also our <Link href="/terms">Terms of Service</Link>.
        </p>
      </section>
    </LegalPage>
  );
}
