# Tech Stack Decisions — Unit 10: Web GUI

## Language & Framework: TypeScript 5 + React 18
- Đã chốt ở Inception (`technology-direction.md`/ADR-0003) — không đổi.

## Build Tool: Vite
- Chuẩn hiện đại cho React/TypeScript SPA, HMR nhanh, cấu hình tối thiểu (đã chốt ở LLD Question 8).

## Testing: Vitest + React Testing Library
- **Ecosystem/library maturity**: Vitest tích hợp native với Vite (dùng chung config, transform pipeline) — tránh cấu hình Babel/ts-jest riêng như Jest truyền thống.
- **Performance**: nhanh hơn Jest cho project Vite (chia sẻ esbuild transform).
- **Team familiarity**: N/A mới với dự án, nhưng API tương thích Jest (dễ chuyển đổi kiến thức).
- **Long-term maintenance**: cộng đồng đang tăng trưởng nhanh, chính thức khuyến nghị bởi Vite team.
- **Licensing/cost**: MIT, mã nguồn mở.

## Package Manager: npm
- Nhất quán, không cần `yarn`/`pnpm` cho 1 project nhỏ.

## Consistency with System-Wide Direction
`technology-direction.md` đã chỉ định React cho Frontend GUI — NFR Requirements xác nhận (không đảo ngược) và chọn tooling cụ thể (Vite, Vitest) chưa được quyết định ở Inception.

## Related ADR
`aidlc-docs/decisions/ADR-0021-web-gui-vite-vitest-toolchain.md`.
