import { createContext, useContext, useReducer, type Dispatch, type ReactNode } from "react";
import type { AuthoringStep } from "../api/client";

/**
 * Bug report: chạy chuỗi AI ở tab 1a rồi chuyển sang 1b/1c trong lúc nó vẫn
 * chạy — mỗi `AuthoringModeBar` giữ `running` cục bộ, nên tab vừa mount không
 * biết chuỗi kia còn dở, nút "Chạy" của nó vẫn bấm được, và Creator bắn thêm
 * một lượt gọi API chồng lên lượt đang chạy. Trạng thái chạy phải sống ở đây
 * — ngoài mọi trang — để tab nào cũng thấy cùng một sự thật: đang chạy hay
 * không, và đang ở bước nào trong chuỗi.
 */
export interface AuthoringRunState {
  running: boolean;
  steps: AuthoringStep[];
  currentIndex: number;
}

export type AuthoringRunAction =
  | { type: "START"; steps: AuthoringStep[] }
  | { type: "PROGRESS"; index: number }
  | { type: "FINISH" };

const initialState: AuthoringRunState = {
  running: false,
  steps: [],
  currentIndex: -1,
};

function reducer(state: AuthoringRunState, action: AuthoringRunAction): AuthoringRunState {
  switch (action.type) {
    case "START":
      return { running: true, steps: action.steps, currentIndex: -1 };
    case "PROGRESS":
      return { ...state, currentIndex: action.index };
    case "FINISH":
      return { ...state, running: false };
  }
}

const AuthoringRunContext = createContext<AuthoringRunState>(initialState);
const AuthoringRunDispatchContext = createContext<Dispatch<AuthoringRunAction>>(() => {});

/**
 * Không lưu localStorage như ProjectDraftContext: đây là trạng thái "đang
 * bay", vô nghĩa sau khi tải lại trang (không có request nào còn chạy để nối
 * lại), nên khởi động lại thành `running: false` là đúng chứ không phải mất
 * dữ liệu.
 */
export function AuthoringRunProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(reducer, initialState);
  return (
    <AuthoringRunContext.Provider value={state}>
      <AuthoringRunDispatchContext.Provider value={dispatch}>{children}</AuthoringRunDispatchContext.Provider>
    </AuthoringRunContext.Provider>
  );
}

export function useAuthoringRun(): AuthoringRunState {
  return useContext(AuthoringRunContext);
}

export function useAuthoringRunDispatch(): Dispatch<AuthoringRunAction> {
  return useContext(AuthoringRunDispatchContext);
}
