They've got, at the moment 4000 listings for rental;
So they are REALLY small;
NVM - Dutch Association of Real Estate Agents;


Funda (funda.nl) is the dominant real estate portal in the Netherlands. Founded in 2001 by the NVM (Dutch Association of Real Estate Agents), it has evolved from a simple listing site into a high-traffic, data-driven tech company.

### **1. Technology Stack Report**

Funda has transitioned from a legacy .NET monolith into a modern, cloud-native microservices architecture.

* **Frontend:** * **Framework:** Moving toward **Vue.js** and **Nuxt.js** (for Server-Side Rendering).
* **Mobile:** Uses **Flutter** for cross-platform app development.
* **Architecture:** Implements a **"Lighthouse Architecture"** and an internal component library (built with **RekaUI**).
* **CMS:** Recently migrated to **Contentful** (Headless CMS) to decouple content from presentation.


* **Backend & Infrastructure:**
* **Languages:** Historically **ASP.NET (.NET Framework)**; modern services are built as containerized **.NET Core/Microservices**.
* **Cloud Provider:** **Microsoft Azure** (migrated from on-premise in 2020).
* **Orchestration:** **Kubernetes (AKS)** managed via **Helm** and **Terraform** (IaC).
* **CI/CD:** Transitioning from Atlassian Bamboo to **GitHub** and **Azure DevOps**.
* **Data & Messaging:** Uses **Azure Service Bus**, **Cosmos DB**, and **Apache Superset** for embedded analytics.


* **APIs:** Does not offer a public API (official data is restricted to NVM members), though unofficial reverse-engineered wrappers (like *pyfunda*) exist for the mobile API.

### **2. Size and Scope**

* **Traffic:** ~14–15 million visits per month (Jan 2026 data), reaching up to 5 million unique visitors. It is the #1 Real Estate site in the Netherlands.
* **User Engagement:** High "stickiness" with an average visit duration of ~5 minutes and ~27–30 pages viewed per session.
* **Workforce:** Approximately **150–200 employees** (referred to as "bright minds" in their engineering blog).
* **Revenue:** Estimated between **$50M – $100M** annually.

### **3. Business and Market Context**

* **Ownership:** Majorly owned by the **NVM** (70% interest historically), but in late 2023, private equity firm **General Atlantic** acquired a significant minority stake for roughly **$103M**, valuing the company at a high multiple.
* **Scope:** While its core is residential sales (`funda.nl`), it also operates `fundainbusiness.nl` for commercial real estate and provides extensive data services to real estate agents.
* **Data Innovation:** They actively use "Big Data" to influence the market, even launching projects like "funda House"—a conceptual home designed entirely based on user search behavior and click patterns.

### **4. Technical Challenges & Evolution**

* **Legacy Migration:** A major ongoing effort is decomposing their 20-year-old .NET monolith into microservices.
* **Bot Protection:** Due to the high value of real estate data, Funda employs significant anti-scraping measures to prevent aggregators from siphoning listings.
* **Performance:** They treat "Performance as a Feature," focusing heavily on Core Web Vitals to maintain their #1 SEO ranking in the Dutch market.

### **Summary Table**

| Category | Detail |
| --- | --- |
| **Core Stack** | Vue.js, Nuxt.js, .NET Core, Flutter |
| **Hosting** | Microsoft Azure (Kubernetes) |
| **Headquarters** | Amsterdam, Piet Heinkade |
| **Monthly Visits** | ~14.6 Million |
| **Market Rank** | #1 Real Estate Portal in NL |
| **Primary Owner** | NVM (with General Atlantic backing) |


# Sources

1. [Decoding Funda’s tech stack (Engineering Blog)](https://blog.funda.nl/decoding-fundas-tech-stack/?utm_source=chatgpt.com)
2. [Funda Tech overview page](https://jobs.funda.nl/l/nl/tech?utm_source=chatgpt.com)
3. [Funda transition to headless CMS (Contentful)](https://blog.funda.nl/a-leap-forward-fundas-journey-transitioning-to-a-headless-cms/?utm_source=chatgpt.com)
4. [Funda company technology profile (Enlyft)](https://enlyft.com/tech/company/funda.nl?utm_source=chatgpt.com)
5. [Technology detection report (Datafragment)](https://www.datafragment.com/technology-lookup/funda.store?utm_source=chatgpt.com)

---

[1]: https://enlyft.com/tech/company/funda.nl?utm_source=chatgpt.com "Funda Technologies Stack and Company Profile"
[2]: https://jobs.funda.nl/l/nl/tech?utm_source=chatgpt.com "Tech bij Funda"
[3]: https://blog.funda.nl/decoding-fundas-tech-stack/?utm_source=chatgpt.com "Decoding funda's tech stack: The reasons behind our choices"
[4]: https://blog.funda.nl/a-leap-forward-fundas-journey-transitioning-to-a-headless-cms/?utm_source=chatgpt.com "Funda’s journey transitioning to a headless CMS"
[5]: https://www.datafragment.com/technology-lookup/funda.store?utm_source=chatgpt.com "Technology profile funda.store - Datafragment"
