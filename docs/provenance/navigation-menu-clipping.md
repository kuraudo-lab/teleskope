# Navigation menus at narrow widths

User acceptance found EKS/Kubernetes menus invisible. In the actual in-app browser the menu opens in DOM but the <=1180px navigation overflow:auto clips it. Preserve menu behavior and allow wrapping rather than clipping dropdown descendants. Verify both groups visually and by pointer hit testing at desktop, current narrow window and mobile widths. Existing wide-screen DOM visibility tests missed clipping; regression must verify that actual menu items receive pointer hits. Regenerate embedded assets and reports, update the current PR and acceptance server.
