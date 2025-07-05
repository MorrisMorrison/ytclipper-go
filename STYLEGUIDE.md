# YTClipper Style Guide

A comprehensive design system and style guide for the YTClipper application.

## 🎨 Design Principles

### Visual Design Philosophy
- **Modern & Clean**: Emphasis on simplicity and clarity
- **Glassmorphism**: Semi-transparent elements with backdrop blur effects
- **Consistent**: Unified design language across all components
- **Accessible**: High contrast ratios and clear focus states
- **Responsive**: Adaptive design that works on all screen sizes

### Key Design Elements
- **Card-based layouts** with subtle shadows and rounded corners
- **Gradient backgrounds** for depth and visual interest
- **Smooth animations** for enhanced user experience
- **Consistent spacing** using CSS custom properties
- **Theme-aware design** supporting both light and dark modes

---

## 🎯 CSS Custom Properties

### Color System

#### Primary Colors
```css
--primary: #38a169        /* Main brand color - Green */
--primary-dark: #2f855a   /* Darker shade for hover states */
--primary-light: #48bb78  /* Lighter shade for accents */
```

#### Secondary Colors
```css
--secondary: #4299e1      /* Blue accent color */
--accent: #10b981         /* Teal accent color */
--success: #38a169        /* Success states */
--warning: #ed8936        /* Warning states */
--error: #e53e3e          /* Error states */
```

#### Dark Theme Colors
```css
--bg-primary: #2d3748     /* Main dark background */
--bg-secondary: #4a5568   /* Secondary dark background */
--bg-surface: #1a202c     /* Surface elements */
--text-primary: #cbd5e0   /* Primary text on dark */
--text-secondary: #718096 /* Secondary text on dark */
--text-muted: #4a5568     /* Muted text on dark */
```

#### Light Theme Colors
```css
--bg-primary-light: #ffffff     /* Main light background */
--bg-secondary-light: #f7fafc   /* Secondary light background */
--bg-surface-light: #edf2f7     /* Light surface elements */
--text-primary-light: #1a202c   /* Primary text on light */
--text-secondary-light: #4a5568 /* Secondary text on light */
--text-muted-light: #718096     /* Muted text on light */
```

### Typography

#### Font System
```css
--font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
```

#### Font Sizes
```css
--font-size-xs: 0.75rem     /* 12px */
--font-size-sm: 0.875rem    /* 14px */
--font-size-base: 1rem      /* 16px */
--font-size-lg: 1.125rem    /* 18px */
--font-size-xl: 1.25rem     /* 20px */
--font-size-2xl: 1.5rem     /* 24px */
--font-size-3xl: 1.875rem   /* 30px */
--font-size-4xl: 2.25rem    /* 36px */
```

### Spacing System
```css
--spacing-xs: 0.25rem      /* 4px */
--spacing-sm: 0.5rem       /* 8px */
--spacing-md: 0.75rem      /* 12px */
--spacing-lg: 1rem         /* 16px */
--spacing-xl: 1.25rem      /* 20px */
--spacing-2xl: 1.5rem      /* 24px */
--spacing-3xl: 2rem        /* 32px */
```

### Border Radius
```css
--border-radius: 0.375rem     /* 6px - Default */
--border-radius-lg: 0.5rem    /* 8px - Large */
--border-radius-xl: 0.75rem   /* 12px - Extra Large */
--border-width: 1px           /* Standard border */
```

### Shadows
```css
--shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
--shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1);
--shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);
```

### Transitions
```css
--transition-fast: 0.15s ease-in-out
--transition-normal: 0.3s ease-in-out
--transition-slow: 0.5s ease-in-out
```

---

## 🧩 Components

### Main Card Container

#### Usage
```html
<div class="main-card">
  <!-- Content -->
</div>
```

#### Specifications
- **Background**: Semi-transparent with glassmorphism effect
- **Padding**: `var(--spacing-3xl)` (32px)
- **Border Radius**: `var(--border-radius-xl)` (12px)
- **Max Width**: 600px
- **Shadow**: Multi-layered shadows with theme variations
- **Hover Effect**: Subtle lift with enhanced glow

#### Visual Effects
- Backdrop blur for glassmorphism
- Gradient accent line at top
- Enhanced shadows on hover
- Smooth transitions

### Form Elements

#### Input Fields
```html
<input class="input" type="text" placeholder="Placeholder text">
```

**Specifications:**
- **Height**: 2.5rem (40px)
- **Padding**: `var(--spacing-sm) var(--spacing-md)` (8px 12px)
- **Border**: 1px solid with theme-aware colors
- **Border Radius**: `var(--border-radius)` (6px)
- **Font**: `var(--font-size-base)` with system font stack

**States:**
- **Default**: Subtle border color
- **Hover**: Enhanced border color with light shadow
- **Focus**: Primary color border with focus ring and lift effect
- **Disabled**: Muted colors with reduced opacity

#### Select Dropdown
```html
<select class="input">
  <option>Option 1</option>
</select>
```

**Special Features:**
- **Custom Arrow**: SVG-based dropdown indicator
- **Disabled State**: Lock icon instead of arrow
- **Theme Support**: Arrow color adapts to theme
- **Consistent Styling**: Matches input field appearance

#### Buttons
```html
<button class="button">Button Text</button>
```

**Specifications:**
- **Background**: Gradient from primary to primary-dark
- **Padding**: `var(--spacing-md) var(--spacing-xl)` (12px 20px)
- **Border Radius**: `var(--border-radius)` (6px)
- **Font Weight**: 500 (medium)

**Visual Effects:**
- Gradient background with subtle highlights
- Shine animation on hover
- Scale transform on interaction
- Enhanced shadows and glow effects

### Progress Indicator

#### Usage
```html
<div class="progress">
  <div class="progress-bar-custom"></div>
</div>
```

**Specifications:**
- **Height**: 0.75rem (12px)
- **Background**: Inset shadow for depth
- **Progress Bar**: Gradient with shimmer animation
- **Border Radius**: `var(--border-radius-lg)` (8px)

**Animations:**
- Width transition for progress updates
- Shimmer effect for visual interest
- Shine overlay for premium feel

### Theme Toggle

#### Switch Component
```html
<label class="switch">
  <input type="checkbox">
  <span class="slider round"></span>
</label>
```

**Specifications:**
- **Size**: 60px × 34px
- **Slider**: 26px diameter circle
- **Animation**: Smooth slide transition
- **Colors**: Theme-aware background colors

---

## 🎭 Theme System

### Light Theme
- **Background**: Subtle gradient from white to light gray
- **Card**: White background with light shadows
- **Text**: Dark text on light backgrounds
- **Borders**: Light gray borders

### Dark Theme
- **Background**: Gradient from dark blue-gray to darker surface
- **Card**: Semi-transparent dark with enhanced shadows
- **Text**: Light text on dark backgrounds
- **Borders**: Dark borders with subtle highlights

### Theme Implementation
```css
/* Light theme (default) */
body {
  background: linear-gradient(135deg, var(--bg-primary-light) 0%, var(--bg-secondary-light) 100%);
  color: var(--text-primary-light);
}

/* Dark theme */
body.dark {
  background: linear-gradient(135deg, var(--bg-primary) 0%, var(--bg-surface) 100%);
  color: var(--text-primary);
}
```

---

## 📱 Responsive Design

### Breakpoints
- **Mobile**: < 768px
- **Tablet**: 768px - 1024px
- **Desktop**: > 1024px

### Responsive Strategy
- **Mobile-first approach**
- **Flexible layouts** using CSS Flexbox
- **Responsive video player** with aspect-ratio
- **Touch-friendly** button sizes (minimum 44px)
- **Adaptive spacing** using CSS custom properties

### Video Player Responsiveness
```css
video {
  width: 100%;
  height: auto;
  max-width: 640px;
  aspect-ratio: 16/9;
}
```

---

## 🎪 Animation Guidelines

### Animation Principles
- **Purposeful**: Animations should enhance UX, not distract
- **Consistent**: Use standardized timing functions
- **Subtle**: Prefer gentle, refined movements
- **Performant**: Use transform and opacity for smooth animations

### Standard Timing
```css
--transition-fast: 0.15s ease-in-out    /* Quick feedback */
--transition-normal: 0.3s ease-in-out   /* Standard transitions */
--transition-slow: 0.5s ease-in-out     /* Dramatic effects */
```

### Common Animations
- **Hover Lift**: `transform: translateY(-1px to -2px)`
- **Button Scale**: `transform: scale(1.01 to 1.02)`
- **Focus Ring**: `box-shadow` with primary color
- **Shimmer Effects**: Background position animation

---

## 🛠️ Implementation Guidelines

### CSS Architecture
- **CSS Custom Properties** for all design tokens
- **Component-based** styling approach
- **Theme-aware** design using CSS classes
- **Progressive enhancement** for better accessibility

### File Structure
```
static/css/
├── styles.css      # Main component styles
└── utilities.css   # Utility classes
```

### Naming Conventions
- **Semantic names** for components (`.main-card`, `.progress`)
- **State-based classes** (`:hover`, `:focus`, `:disabled`)
- **Theme variants** (`body.dark .component`)
- **Utility classes** for spacing and layout

### Best Practices
1. **Use CSS custom properties** for consistency
2. **Follow the spacing system** for visual rhythm
3. **Implement proper focus states** for accessibility
4. **Test both themes** during development
5. **Maintain animation performance** with transform/opacity
6. **Ensure responsive behavior** across devices

---

## 🔧 Development Workflow

### Adding New Components
1. **Define in design system** - Add CSS custom properties if needed
2. **Create component styles** - Follow existing patterns
3. **Implement theme support** - Test both light and dark modes
4. **Add interactions** - Hover, focus, and active states
5. **Test responsiveness** - Ensure mobile compatibility
6. **Document usage** - Update this style guide

### Modifying Existing Components
1. **Check design system** - Use existing tokens when possible
2. **Maintain consistency** - Follow established patterns
3. **Update both themes** - Ensure changes work in all modes
4. **Test interactions** - Verify all states still work
5. **Update documentation** - Keep style guide current

---

## 📋 Component Checklist

### Required States for Interactive Elements
- [ ] Default state
- [ ] Hover state
- [ ] Focus state (for accessibility)
- [ ] Active/pressed state
- [ ] Disabled state (when applicable)
- [ ] Loading state (when applicable)

### Theme Compatibility
- [ ] Light theme styling
- [ ] Dark theme styling
- [ ] Smooth theme transitions
- [ ] Consistent behavior across themes

### Accessibility Requirements
- [ ] Sufficient color contrast (WCAG 2.1 AA)
- [ ] Keyboard navigation support
- [ ] Screen reader compatibility
- [ ] Focus indicators
- [ ] Touch target sizes (44px minimum)

---

This style guide serves as the single source of truth for all design decisions in the YTClipper project. Follow these guidelines to maintain visual consistency and create a cohesive user experience.