# ConceptFlow UI - TODO List

## 📱 Mobile Optimizations (Future Work)

### High Priority

#### 1. Step Pills Mobile Layout
**Current Issue:** 5 step pills wrap awkwardly on mobile screens (< 640px), causing layout issues and making it hard to identify the current step.

**Proposed Solutions:**
- **Option A:** Show only icon + number on mobile (remove text labels)
- **Option B:** Implement horizontal scroll for pills
- **Option C:** Replace pills with "Bước X/5" text indicator

**Files to Modify:**
- `src/components/AppShell.module.css` (@media 640px section)
- `src/components/AppShell.tsx` (conditional rendering logic)

**Acceptance Criteria:**
- All 5 steps visible and identifiable on mobile
- Current step clearly highlighted
- No wrapping or overlapping elements
- Touch-friendly tap targets (minimum 44x44px)

---

#### 2. Progress Tracker Mobile Display
**Current Issue:** ProgressTracker displays 6-8 pipeline steps, consuming excessive vertical space on mobile.

**Proposed Solutions:**
- Collapse completed steps (show only checkmark)
- Display only current + next 2 steps
- Alternative: Horizontal progress bar with step count

**Files to Modify:**
- `src/components/ProgressTracker.tsx`
- `src/components/ProgressTracker.module.css`

**Acceptance Criteria:**
- Compact display fitting within viewport
- Still shows progress clearly
- User can expand to see all steps if needed

---

### Medium Priority

#### 3. Script Editor Mobile Layout
**Current Issue:** Two-column layout (ScriptAssistant + ScriptEditor) doesn't collapse gracefully on narrow screens.

**Files to Check:**
- `src/pages/ScriptStepPage.tsx`
- `src/pages/WizardSteps.module.css` (.scriptLayout)

**Acceptance Criteria:**
- Single column layout on mobile
- Adequate touch targets for buttons
- Comfortable typing experience in textarea

---

#### 4. Form Input Touch Optimization
**Current Issue:** Input fields and buttons may not have optimal touch targets on mobile.

**Areas to Review:**
- All text inputs (minimum 44px height)
- Buttons (minimum 44x44px touch area)
- Select/dropdown controls
- Toggle switches

**Files to Audit:**
- `src/styles/glass.module.css` (input styles)
- All component-specific CSS files

---

#### 5. Video Player Responsive Controls
**Current Issue:** Native HTML5 video controls may be too small on mobile.

**Considerations:**
- Native controls should be adequate on most devices
- Consider custom controls if user feedback indicates issues

**Files:**
- `src/components/VideoPlayer.tsx`
- `src/components/VideoPlayer.module.css`

---

### Low Priority

#### 6. Glassmorphism Performance on Low-End Devices
**Current Issue:** `backdrop-filter: blur()` can be expensive on older mobile devices.

**Proposed Solutions:**
- Detect device performance (via CSS @media (prefers-reduced-motion) or JS)
- Fall back to solid backgrounds on low-end devices
- Consider reducing blur radius on mobile

**Files:**
- `src/styles/glass.module.css`
- `src/styles/theme.css`

---

#### 7. Dark Mode Transition Smoothness
**Current Issue:** Theme transitions may be janky on some mobile browsers.

**Investigation Needed:**
- Test on iOS Safari, Chrome Mobile, Firefox Mobile
- Consider using `content-visibility` for off-screen content
- May need to reduce transition complexity

**Files:**
- `src/context/ThemeContext.tsx`
- `src/styles/theme.css`

---

## 🎯 Desktop UX Improvements (Nice to Have)

### 1. Wizard Step Navigation
**Feature:** Allow clicking on completed steps to jump back.

**Current Behavior:** Must use "Quay lại" button sequentially.

**Benefit:** Power users can quickly review/edit previous choices.

**Files:**
- `src/components/AppShell.tsx` (make step pills clickable)
- `src/pages/*StepPage.tsx` (add navigation logic)

---

### 2. Keyboard Shortcuts
**Feature:** Global keyboard shortcuts for common actions.

**Examples:**
- `Ctrl+Enter`: Submit current form
- `Esc`: Close modals
- `Ctrl+K`: Focus search/command palette (future)

**Implementation:**
- Create `useKeyboardShortcuts` hook
- Add visual indicators (tooltips showing shortcuts)

---

### 3. Drag & Drop File Upload
**Feature:** Drag script .py files directly into ScriptEditor.

**Current:** Must click "Nhập từ file" button.

**Files:**
- `src/components/ScriptEditor.tsx`

---

## 🐛 Known Minor Issues

### 1. Subtitle Mode "Both" Confusion
**Issue:** UI warns that "both" mode shows duplicate text but still allows selection.

**Solution:** 
- Disable option with tooltip explaining why
- Or: Show prominent warning banner
- Or: Auto-suggest best option based on use case

**Files:**
- Components handling subtitle mode picker (needs identification)

---

### 2. Modal Stack Management
**Issue:** Multiple modals open simultaneously may have z-index conflicts.

**Current Status:** ConfirmModal uses z-index: 9999, should be sufficient.

**Monitor:** If additional modals added, implement modal stack manager.

---

## 📊 Testing Gaps

### Mobile Testing Checklist
- [ ] Test on real iOS devices (Safari)
- [ ] Test on real Android devices (Chrome)
- [ ] Test on tablets (iPad, Android tablets)
- [ ] Test with device simulators (Chrome DevTools, Xcode Simulator)
- [ ] Test landscape orientation
- [ ] Test with system font scaling (accessibility)
- [ ] Test with system zoom (accessibility)

### Browser Compatibility
- [ ] Safari (desktop & mobile)
- [ ] Firefox (desktop & mobile)
- [ ] Chrome (desktop & mobile)
- [ ] Edge (desktop)

### Accessibility Audit
- [ ] Screen reader testing (NVDA, JAWS, VoiceOver)
- [ ] Keyboard-only navigation
- [ ] Color contrast validation (WCAG AA)
- [ ] Focus indicators visible on all interactive elements

---

## 🔧 Technical Debt

### 1. CSS Module Consolidation
**Issue:** Some shared styles duplicated across multiple `.module.css` files.

**Solution:** Extract common patterns to shared modules or CSS custom properties.

**Examples:**
- Button styles (primary, ghost, danger variants)
- Input field styles
- Card/panel layouts

---

### 2. Type Safety for API Responses
**Issue:** Some API responses use `any` or are not fully typed.

**Files to Review:**
- `src/api/client.ts`
- `src/types/index.ts`

---

### 3. Error Boundary Implementation
**Status:** Not implemented.

**Benefit:** Graceful degradation when React components throw.

**Implementation:**
- Create `ErrorBoundary` component
- Wrap App or routes
- Log errors to console/analytics

---

## 📝 Documentation Needs

### 1. Component Documentation
**Status:** Most components have JSDoc comments, but could be more comprehensive.

**Improvements:**
- Add usage examples in JSDoc
- Document prop types with descriptions
- Add Storybook (optional, future)

---

### 2. Accessibility Documentation
**Status:** Accessibility features implemented but not documented for developers.

**Needed:**
- ARIA patterns reference
- Keyboard interaction guide
- Screen reader testing guide

---

## 🚀 Performance Optimizations (Future)

### 1. Code Splitting
**Status:** Not implemented (Vite provides basic splitting).

**Opportunities:**
- Lazy load route components
- Lazy load heavy dependencies (video player, markdown renderer)

---

### 2. Image Optimization
**Current:** PNG logo at 192px.

**Improvements:**
- WebP format with PNG fallback
- Multiple sizes for different screen densities
- Lazy loading for off-screen images

---

### 3. Bundle Size Analysis
**Action Needed:** Run `vite-bundle-visualizer` to identify large dependencies.

**Potential Targets:**
- React Router (may be able to use lighter alternative)
- Any unused dependencies

---

## 📅 Maintenance Schedule

### Regular Reviews
- **Monthly:** Review and prioritize TODO items
- **Quarterly:** Accessibility audit
- **Bi-annually:** Dependencies update and security audit

### Version Milestones
- **v1.1:** Complete high-priority mobile optimizations
- **v1.2:** Implement desktop UX improvements
- **v2.0:** Major refactor (if needed based on usage feedback)

---

## 📬 User Feedback Integration

### Feedback Collection Points
- In-app feedback button (future)
- GitHub Issues (if open source)
- User interviews/surveys

### Common Request Tracking
- [ ] Mobile experience complaints
- [ ] Dark mode issues
- [ ] Performance on slow connections
- [ ] Browser-specific bugs

---

**Last Updated:** 2024-01-XX  
**Maintained By:** Development Team  
**Review Frequency:** Monthly
