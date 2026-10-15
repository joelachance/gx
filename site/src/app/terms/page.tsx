import type { Metadata } from "next";
import Link from "next/link";
import { LegalPage } from "@/components/legal-page";
import { SUPPORT_EMAIL, SUPPORT_EMAIL_URL } from "@/lib/site-links";

export const metadata: Metadata = {
  title: "Terms of Service — gx",
  description: "Terms of Service for gx by Satori Engineering Co.",
};

export default function TermsPage() {
  return (
    <LegalPage title="Terms of Service" updated="July 14, 2026">
      <section>
        <h2>1. Agreement</h2>
        <p>
          These Terms of Service (“Terms”) govern your access to and use of gx
          and related websites, apps, and services (the “Service”) provided by
          Satori Engineering Co. (“Satori,” “we,” “us,” or “our”). By creating
          an account or using the Service, you agree to these Terms.
        </p>
      </section>

      <section>
        <h2>2. The Service</h2>
        <p>
          gx is a code review and verification product. Features may change over
          time. We may add, modify, or discontinue parts of the Service with or
          without notice, except where required by law or a paid subscription
          agreement.
        </p>
      </section>

      <section>
        <h2>3. Accounts</h2>
        <p>
          You must provide accurate account information and keep your
          credentials secure. You are responsible for activity under your
          account. Notify us promptly at{" "}
          <a href={SUPPORT_EMAIL_URL}>{SUPPORT_EMAIL}</a> if you suspect
          unauthorized access.
        </p>
      </section>

      <section>
        <h2>4. Acceptable use</h2>
        <p>You agree not to:</p>
        <ul>
          <li>Use the Service for unlawful purposes</li>
          <li>
            Attempt to access other users’ data or systems without authorization
          </li>
          <li>
            Interfere with or disrupt the Service, including through abuse,
            scraping that harms availability, or reverse engineering except as
            allowed by law
          </li>
          <li>
            Upload malware or content you do not have rights to provide
          </li>
        </ul>
      </section>

      <section>
        <h2>5. Your content</h2>
        <p>
          You retain ownership of code, comments, and other materials you submit
          (“Your Content”). You grant Satori a limited license to host, process,
          display, and otherwise use Your Content solely to operate and improve
          the Service for you. You represent that you have the rights needed to
          provide Your Content.
        </p>
      </section>

      <section>
        <h2>6. Subscriptions and trials</h2>
        <p>
          Paid plans, trials, and billing are described at purchase or in-product.
          Fees are non-refundable except where required by law or stated
          otherwise. We may change prices with reasonable notice for renewals.
          Failure to pay may result in suspension or termination of paid
          features.
        </p>
      </section>

      <section>
        <h2>7. Intellectual property</h2>
        <p>
          The Service, including software, branding, and documentation, is owned
          by Satori or its licensors. These Terms do not transfer any ownership
          to you other than the limited right to use the Service as permitted
          here.
        </p>
      </section>

      <section>
        <h2>8. Third-party services</h2>
        <p>
          The Service may integrate with third parties (for example GitHub,
          payment processors, or analytics providers). Their terms and privacy
          policies apply to your use of those services. We are not responsible
          for third-party offerings outside our control.
        </p>
      </section>

      <section>
        <h2>9. Disclaimers</h2>
        <p>
          THE SERVICE IS PROVIDED “AS IS” AND “AS AVAILABLE.” TO THE MAXIMUM
          EXTENT PERMITTED BY LAW, SATORI DISCLAIMS ALL WARRANTIES, EXPRESS OR
          IMPLIED, INCLUDING MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE,
          AND NON-INFRINGEMENT. We do not warrant that the Service will be
          uninterrupted, error-free, or that reviews or analysis will catch every
          issue in your code.
        </p>
      </section>

      <section>
        <h2>10. Limitation of liability</h2>
        <p>
          TO THE MAXIMUM EXTENT PERMITTED BY LAW, SATORI AND ITS AFFILIATES WILL
          NOT BE LIABLE FOR INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR
          PUNITIVE DAMAGES, OR ANY LOSS OF PROFITS, DATA, OR GOODWILL. OUR TOTAL
          LIABILITY FOR CLAIMS ARISING OUT OF THESE TERMS OR THE SERVICE WILL
          NOT EXCEED THE AMOUNTS YOU PAID US FOR THE SERVICE IN THE TWELVE (12)
          MONTHS BEFORE THE CLAIM.
        </p>
      </section>

      <section>
        <h2>11. Termination</h2>
        <p>
          You may stop using the Service at any time. We may suspend or terminate
          access if you violate these Terms or if we discontinue the Service. Upon
          termination, your right to use the Service ends. Provisions that by
          their nature should survive will survive.
        </p>
      </section>

      <section>
        <h2>12. Privacy</h2>
        <p>
          Our collection and use of personal information is described in our{" "}
          <Link href="/privacy">Privacy Policy</Link>.
        </p>
      </section>

      <section>
        <h2>13. Changes</h2>
        <p>
          We may update these Terms from time to time. We will post the updated
          Terms on this page and revise the “Last updated” date. Continued use
          after changes become effective constitutes acceptance of the updated
          Terms.
        </p>
      </section>

      <section>
        <h2>14. Contact</h2>
        <p>
          Questions about these Terms:{" "}
          <a href={SUPPORT_EMAIL_URL}>{SUPPORT_EMAIL}</a>
        </p>
      </section>
    </LegalPage>
  );
}
