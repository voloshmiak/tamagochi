import { useEffect, useRef } from "react";
import { useUI } from "../store/ui";

const STREAM = "https://listen.reyfm.de/lofi_128kbps.mp3";

export function MusicPlayer() {
  const sound = useUI((s) => s.sound);
  const volume = useUI((s) => s.volume);
  const audio = useRef<HTMLAudioElement | null>(null);

  useEffect(() => {
    if (audio.current) audio.current.volume = volume;
  }, [volume]);

  useEffect(() => {
    if (!sound) {

      if (audio.current) {
        audio.current.pause();
        audio.current.removeAttribute("src");
        audio.current.load();
        audio.current = null;
      }
      return;
    }


    const a = new Audio(STREAM);
    a.volume = useUI.getState().volume;
    audio.current = a;

    const start = () => void a.play().catch(() => {});
    a.play().catch(() => {
      window.addEventListener("pointerdown", start, { once: true });
      window.addEventListener("keydown", start, { once: true });
    });

    return () => {
      window.removeEventListener("pointerdown", start);
      window.removeEventListener("keydown", start);
      a.pause();
      a.removeAttribute("src");
      a.load();
    };
  }, [sound]);

  return null;
}