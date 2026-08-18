---
name: Institutional Ledger
colors:
  surface: '#f7f9fb'
  surface-dim: '#d8dadc'
  surface-bright: '#f7f9fb'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f2f4f6'
  surface-container: '#eceef0'
  surface-container-high: '#e6e8ea'
  surface-container-highest: '#e0e3e5'
  on-surface: '#191c1e'
  on-surface-variant: '#45464d'
  inverse-surface: '#2d3133'
  inverse-on-surface: '#eff1f3'
  outline: '#76777d'
  outline-variant: '#c6c6cd'
  surface-tint: '#565e74'
  primary: '#000000'
  on-primary: '#ffffff'
  primary-container: '#131b2e'
  on-primary-container: '#7c839b'
  inverse-primary: '#bec6e0'
  secondary: '#516072'
  on-secondary: '#ffffff'
  secondary-container: '#d2e1f7'
  on-secondary-container: '#556477'
  tertiary: '#000000'
  on-tertiary: '#ffffff'
  tertiary-container: '#271901'
  on-tertiary-container: '#98805d'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#dae2fd'
  primary-fixed-dim: '#bec6e0'
  on-primary-fixed: '#131b2e'
  on-primary-fixed-variant: '#3f465c'
  secondary-fixed: '#d4e4fa'
  secondary-fixed-dim: '#b9c8de'
  on-secondary-fixed: '#0d1c2d'
  on-secondary-fixed-variant: '#39485a'
  tertiary-fixed: '#fcdeb5'
  tertiary-fixed-dim: '#dec29a'
  on-tertiary-fixed: '#271901'
  on-tertiary-fixed-variant: '#574425'
  background: '#f7f9fb'
  on-background: '#191c1e'
  surface-variant: '#e0e3e5'
typography:
  display-lg:
    fontFamily: Public Sans
    fontSize: 32px
    fontWeight: '700'
    lineHeight: 40px
    letterSpacing: -0.02em
  headline-md:
    fontFamily: Public Sans
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
    letterSpacing: -0.01em
  headline-sm:
    fontFamily: Public Sans
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
  body-lg:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  body-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  body-sm:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
  label-bold:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
    letterSpacing: 0.05em
  mono-data:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  base: 4px
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
  gutter: 16px
  margin-mobile: 16px
  margin-desktop: 32px
  max-width: 1440px
---

## Brand & Style

This design system is engineered for high-stakes financial operations, focusing on the management of digital assets with an institutional-grade rigor. The brand personality is rooted in **precision, stability, and absolute transparency**. It targets compliance officers, asset managers, and financial engineers who require a high-density information environment that minimizes cognitive load while maximizing trust.

The visual style is **refined Minimalism with a Corporate-Modern backbone**. It avoids decorative flourishes in favor of structural integrity. Hierarchy is established through meticulous spacing and subtle tonal shifts rather than vibrant color. Every element is designed to feel "locked-in" and intentional, evoking the reliability of a physical vault combined with the efficiency of a high-frequency trading terminal. The interface should feel crisp, quiet, and authoritative.

## Colors

The color strategy utilizes a restricted palette to maintain a serious, institutional tone. 

- **Primary (Deep Navy):** Reserved for core navigation, primary actions, and high-level structural headings. It represents the foundation of the platform.
- **Secondary (Slate):** Used for supporting text, icons, and non-critical interactive elements. It provides a sophisticated bridge between primary content and the background.
- **Functional (Green/Amber):** These are applied with high restraint. Green indicates cleared transactions or compliant states; Amber is used for encumbrances, pending approvals, or warnings.
- **Background & Surfaces:** A base of Crisp White (#FFFFFF) is used for data containers to ensure maximum legibility, while Light Gray (#F8FAFC) is used for the application background to create a subtle sense of depth and separation between the workspace and the content cards.

## Typography

The system utilizes a dual-sans approach to balance institutional authority with technical utility. **Public Sans** is used for headlines to provide a sturdy, official, and slightly more characterful presence. **Inter** is used for all UI components, body text, and data-heavy views due to its exceptional legibility and systematic design.

For financial data, always enable **tabular numerals** (`tnum`) to ensure that columns of numbers align perfectly for easy comparison. The `label-bold` style should be used for table headers and section labels to provide clear boundaries without requiring heavy background colors.

## Layout & Spacing

This design system employs a **12-column fixed grid** for desktop views, centered within a maximum width of 1440px. The layout philosophy is "Data First," meaning margins and gutters are kept lean to maximize the real estate available for tables and charts.

- **Desktop (1024px+):** 12 columns, 16px gutters, 32px side margins.
- **Tablet (768px - 1023px):** 8 columns, 16px gutters, 24px side margins.
- **Mobile (Up to 767px):** 4 columns, 12px gutters, 16px side margins.

Horizontal spacing between related data points should follow a 4px (base) scale to maintain high density while ensuring optical separation. Vertical spacing between distinct sections should be generous (32px) to allow the user's eyes to rest.

## Elevation & Depth

To maintain a professional, flat aesthetic, this design system avoids heavy shadows. Instead, it uses **Tonal Layers** and **Crisp Outlines** to communicate hierarchy.

- **Level 0 (Background):** #F8FAFC. The foundation of the application.
- **Level 1 (Cards/Panels):** #FFFFFF. White surfaces with a 1px solid border (#E2E8F0). No shadow. This is the primary work surface.
- **Level 2 (Modals/Dropdowns):** #FFFFFF. These elements use a subtle, sharp shadow (0px 4px 12px rgba(15, 23, 42, 0.08)) combined with the standard 1px border.
- **Dividers:** Use 1px #F1F5F9 for internal card separations (e.g., table rows).

Interactive depth is achieved through slight background color shifts (e.g., a button moving from Navy to Charcoal on hover) rather than physical lift.

## Shapes

The shape language is **Soft (0.25rem)**. This slight rounding takes the "edge" off the industrial look, making the software feel modern and accessible without losing its serious, rectilinear character.

- **Input Fields & Small Buttons:** 4px (0.25rem) radius.
- **Cards & Larger Containers:** 8px (0.5rem) radius.
- **Status Tags (Chips):** 4px (0.25rem) radius to match the systematic rigor; avoid full pills as they appear too casual for institutional finance.

## Components

### Buttons
- **Primary:** Deep Navy (#0F172A) background, White text. No border.
- **Secondary:** White background, 1px Border (#E2E8F0), Navy text.
- **Ghost:** Transparent background, Slate text. Used for less frequent actions.

### Input Fields
Inputs must have a 1px border (#E2E8F0). On focus, the border changes to Primary Navy with a subtle 2px outer glow in a translucent navy (rgba(15, 23, 42, 0.05)). Placeholder text should be in Slate Blue (#94A3B8).

### Data Tables
The core of the experience. Headers are #F8FAFC with `label-bold` typography. Rows are White with 1px bottom borders. Hover states on rows should use a very faint gray (#F1F5F9) to assist with tracking data across columns.

### Status Chips
Minimalist indicators. 
- **Active:** Light Green background (10% opacity of #10B981) with Solid Green text.
- **Warning:** Light Amber background (10% opacity of #D97706) with Solid Amber text.

### Cards
Cards are the primary container. They should always have a 1px #E2E8F0 border and 0px shadow. If the content is actionable, the border may darken to #CBD5E1 on hover.